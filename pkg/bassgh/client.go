package bassgh

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-github/v43/github"
	"github.com/mattn/go-colorable"
	"github.com/vito/bass-loop/pkg/blobs"
	"github.com/vito/bass-loop/pkg/models"
	"github.com/vito/bass-loop/pkg/runs"
	"github.com/vito/bass/pkg/bass"
	"github.com/vito/bass/pkg/cli"
	"github.com/vito/progrock"
)

type Client struct {
	ExternalURL *url.URL
	DB          models.DB
	Blobs       *blobs.Bucket
	GH          *github.Client
	Sender      *github.User
	Repo        *github.Repository
}

func (client *Client) Scope() *bass.Scope {
	ghscope := bass.NewEmptyScope()
	ghscope.Set("start-check",
		bass.Func("start-check", "[thunk name sha]", client.StartCheck))

	return ghscope
}

func (client *Client) StartCheck(ctx context.Context, thunk bass.Thunk, checkName, sha string) (bass.Combiner, error) {
	run, err := models.CreateThunkRun(ctx, client.DB, client.Sender, thunk)
	if err != nil {
		return nil, fmt.Errorf("create thunk run: %w", err)
	}

	thunkURL, err := client.ExternalURL.Parse("/thunks/" + thunk.Name())
	if err != nil {
		return nil, fmt.Errorf("create thunk run: %w", err)
	}

	runURL, err := client.ExternalURL.Parse("/runs/" + run.ID)
	if err != nil {
		return nil, fmt.Errorf("create thunk run: %w", err)
	}

	output := &github.CheckRunOutput{
		Title: github.String("(" + thunk.Cmd.ToValue().String() + ")"),
		Summary: github.String(strings.Join([]string{
			`* **thunk** [` + thunk.Name() + `](` + thunkURL.String() + `)`,
			`* **run** [` + run.ID + `](` + runURL.String() + `)`,
			``,
			"```sh",
			"# final command",
			thunk.Cmdline(),
			"```",
		}, "\n")),
	}

	checkRun, _, err := client.GH.Checks.CreateCheckRun(ctx, client.Repo.GetOwner().GetLogin(), client.Repo.GetName(), github.CreateCheckRunOptions{
		Name:       checkName,
		HeadSHA:    sha,
		Status:     github.String("in_progress"),
		StartedAt:  &github.Timestamp{Time: time.Now()},
		ExternalID: github.String(run.ID),
		DetailsURL: github.String(runURL.String()),
		Output:     output,
	})
	if err != nil {
		return nil, fmt.Errorf("create check run: %w", err)
	}

	progress := cli.NewProgress()
	thunkCtx := progrock.RecorderToContext(ctx, progrock.NewRecorder(progress))

	return thunk.Start(thunkCtx, bass.Func("handler", "[ok?]", func(ctx context.Context, ok bool) error {
		if err := runs.Record(ctx, client.DB, client.Blobs, run, progress, ok); err != nil {
			return fmt.Errorf("failed to complete: %w", err)
		}

		outBuf := new(bytes.Buffer)
		progress.Summarize(colorable.NewNonColorable(outBuf))
		output.Text = github.String("```\n" + outBuf.String() + "\n```")

		var conclusion string
		if ok {
			conclusion = "success"
		} else if thunkCtx.Err() != nil {
			conclusion = "cancelled"
		} else {
			conclusion = "failure"
		}

		_, _, err := client.GH.Checks.UpdateCheckRun(
			ctx,
			client.Repo.GetOwner().GetLogin(),
			client.Repo.GetName(),
			checkRun.GetID(),
			github.UpdateCheckRunOptions{
				Name:        checkName,
				Status:      github.String("completed"),
				Conclusion:  github.String(conclusion),
				CompletedAt: &github.Timestamp{Time: run.EndTime.Time()},
				Output:      output,
			},
		)
		if err != nil {
			return fmt.Errorf("update check run: %w", err)
		}

		if ok {
			return nil
		}

		// bubble up an error so it gets logged
		//
		// might make sense to remove this someday, but I would rather start with
		// too much logging
		return fmt.Errorf("%s: check %s: %s failed", sha, checkName, thunk)
	}))
}
