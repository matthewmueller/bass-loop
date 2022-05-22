package github

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v43/github"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass/pkg/bass"
	"github.com/vito/bass/pkg/cli"
	"github.com/vito/bass/pkg/runtimes"
	"github.com/vito/bass/pkg/zapctx"
	"github.com/vito/progrock"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var defaultConfigTODO = bass.Config{
	Runtimes: []bass.RuntimeConfig{
		{
			Platform: bass.LinuxPlatform,
			Runtime:  runtimes.BuildkitName,
		},
	},
}

type WebhookHandler struct {
	DB            *sql.DB
	RunCtx        context.Context
	WebhookSecret string
	AppsTransport *ghinstallation.AppsTransport
	Dispatches    *errgroup.Group
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	eventType := r.Header.Get("X-GitHub-Event")
	deliveryID := r.Header.Get("X-GitHub-Delivery")

	if eventType == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "missing event type")
		return
	}

	payloadBytes, err := github.ValidatePayload(r, []byte(h.WebhookSecret))
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprintln(w, "missing event type")
		return
	}

	err = h.Handle(ctx, eventType, deliveryID, payloadBytes)
	if err != nil {
		cli.WriteError(ctx, err)

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err.Error())
		return
	}
}

type RepoEvent struct {
	Repo         *github.Repository   `json:"repository,omitempty"`
	Installation *github.Installation `json:"installation,omitempty"`
}

func (h *WebhookHandler) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	logger := zapctx.FromContext(ctx).With(zap.String("event", eventType), zap.String("delivery", deliveryID))

	logger.Info("handling")

	var event RepoEvent
	err := json.Unmarshal(payload, &event)
	if err != nil {
		return err
	}

	if event.Repo != nil && event.Installation != nil {
		return h.dispatch(
			ctx,
			event.Installation.GetID(),
			event.Repo.GetOwner().GetLogin(),
			event.Repo.GetName(),
			eventType,
			deliveryID,
			payload,
		)
	} else {
		logger.Warn("ignoring unknown event")
	}

	return nil
}

func (h *WebhookHandler) dispatch(ctx context.Context, instID int64, user, repo string, eventType, deliveryID string, payload []byte) error {
	logger := zapctx.FromContext(ctx)

	runCtx := bass.ForkTrace(h.RunCtx)

	var payloadScope *bass.Scope
	err := json.Unmarshal(payload, &payloadScope)
	if err != nil {
		return fmt.Errorf("payload->scope: %w", err)
	}

	scope := bass.NewStandardScope()
	scope.Set("*delivery-id*", bass.String(deliveryID))
	scope.Set("*event*", bass.String(eventType))
	scope.Set("*payload*", payloadScope)
	scope.Set("*github*", BassGitHubClient{
		DB: h.DB,
		GH: github.NewClient(&http.Client{
			Transport: ghinstallation.NewFromAppsTransport(h.AppsTransport, instID),
		}),
		User: user,
		Repo: repo,
	}.Scope())

	// TODO: db-backed pool that looks up user's workers
	pool, err := runtimes.NewPool(&defaultConfigTODO)
	if err != nil {
		return fmt.Errorf("pool: %w", err)
	}
	runCtx = bass.WithRuntimePool(runCtx, pool)

	h.Dispatches.Go(func() error {
		_, err = bass.EvalString(runCtx, scope, `
		(use (.git (linux/alpine/git)))

		(let [{:repository
					 {:clone-url url
						:default-branch branch
						:pushed-at pushed-at}} *payload*
					sha (git:ls-remote url branch pushed-at)
					src (git:checkout url sha)
					project (load (src/project))]
			(project:github-event *payload* *event* *github*))
	`)
		if err != nil {
			logger.Error("delivery failed", zap.String("delivery", deliveryID), zap.Error(err))
			cli.WriteError(runCtx, err)
			return err
		}

		return err
	})

	return nil
}

type BassGitHubClient struct {
	DB         *sql.DB
	GH         *github.Client
	User, Repo string
}

func (client BassGitHubClient) Scope() *bass.Scope {
	ghscope := bass.NewEmptyScope()
	ghscope.Set("start-check",
		bass.Func("start-check", "[thunk name sha]", client.StartCheck))

	return ghscope
}

func (client BassGitHubClient) StartCheck(ctx context.Context, thunk bass.Thunk, name, sha string) (bass.Combiner, error) {
	logger := zapctx.FromContext(ctx)

	sha2, err := thunk.SHA256()
	if err != nil {
		return nil, err
	}

	thunkRun, err := models.CreateThunkRun(ctx, client.DB, sha2)
	if err != nil {
		return nil, fmt.Errorf("create thunk run: %w", err)
	}

	run, _, err := client.GH.Checks.CreateCheckRun(ctx, client.User, client.Repo, github.CreateCheckRunOptions{
		Name:      name,
		HeadSHA:   sha,
		Status:    github.String("in_progress"),
		StartedAt: &github.Timestamp{Time: time.Now()},
	})
	if err != nil {
		return nil, fmt.Errorf("create check run: %w", err)
	}

	progress := cli.NewProgress()
	thunkCtx := progrock.RecorderToContext(ctx, progrock.NewRecorder(progress))

	return thunk.Start(thunkCtx, bass.Func("handler", "[ok?]", func(ctx context.Context, ok bool) error {
		completedAt := time.Now()

		thunkRun.EndTime = sql.NullInt64{
			Int64: completedAt.UnixNano(),
			Valid: true,
		}

		var conclusion string
		if ok {
			thunkRun.Succeeded = sql.NullInt64{Int64: 1, Valid: true}
			conclusion = "success"
		} else if ctx.Err() != nil {
			thunkRun.Succeeded = sql.NullInt64{Int64: 0, Valid: true}
			conclusion = "cancelled"
		} else {
			thunkRun.Succeeded = sql.NullInt64{Int64: 0, Valid: true}
			conclusion = "failure"
		}

		err = progress.EachVertex(func(v *cli.Vertex) error {
			var startTime, endTime sql.NullInt64
			if v.Started != nil {
				startTime.Int64 = v.Started.UnixNano()
				startTime.Valid = true
			}
			if v.Completed != nil {
				endTime.Int64 = v.Completed.UnixNano()
				endTime.Valid = true
			}

			var vErr sql.NullString
			if v.Error != "" {
				vErr.String = v.Error
				vErr.Valid = true
			}

			var cached int
			if v.Cached {
				cached = 1
			}

			vtx := models.Vertex{
				Digest:    v.Digest.String(),
				RunID:     thunkRun.ID,
				Name:      v.Name,
				StartTime: startTime,
				EndTime:   endTime,
				Error:     vErr,
				Cached:    cached,
				Logs:      v.Log.Bytes(),
			}

			err := vtx.Save(ctx, client.DB)
			if err != nil {
				return err
			}

			for _, input := range v.Inputs {
				edge := models.VertexEdge{
					SourceDigest: input.String(),
					TargetDigest: v.Digest.String(),
				}

				err := edge.Insert(ctx, client.DB)
				if err != nil {
					logger.Warn("insert edge", zap.Error(err))
				}
			}

			return nil
		})
		if err != nil {
			cli.WriteError(ctx, err)
			return fmt.Errorf("store vertex logs: %w", err)
		}

		err = thunkRun.Update(ctx, client.DB)
		if err != nil {
			cli.WriteError(ctx, err)
			return fmt.Errorf("update thunk run: %w", err)
		}

		_, _, err := client.GH.Checks.UpdateCheckRun(ctx, client.User, client.Repo, run.GetID(), github.UpdateCheckRunOptions{
			Name:        name,
			Status:      github.String("completed"),
			Conclusion:  github.String(conclusion),
			CompletedAt: &github.Timestamp{Time: completedAt},
		})
		if err != nil {
			cli.WriteError(ctx, err)
			return fmt.Errorf("update check run: %w", err)
		}

		return nil
	}))
}
