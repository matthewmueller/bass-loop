<script>
  import Base16Options from './Base16Options.svelte';
  import { style } from './DefaultStyle.js';
</script>

<svelte:head>
  {@html `<style id="default-theme">:root {${style.vars.join(";")}}</style>`}

  <script type="module">
    // do this in <head> so the style gets set ASAP (preventing flicker)
    import { setStyleIfSet } from "/js/switcher.js";
    setStyleIfSet();
  </script>
</svelte:head>

<div class="choose-theme" id="choosetheme">
  <select id="styleswitcher" value={style.scheme} data-default={style.scheme}>
    <Base16Options />
  </select>

  <script type="module">
    import { switchStyle } from "/js/switcher.js";
    document.getElementById("styleswitcher").onchange = switchStyle;
  </script>
</div>

<style>
  .choose-theme {
    display: flex;
    flex-direction: row-reverse;
    justify-content: flex-start;
  }

  select, :global(.reset) {
    margin-left: 1ch;
  }

  select, :global(.reset) {
    font-family: var(--monospace-font);
    font-size: inherit;
    padding: 5px;
    border-radius: var(--button-radius);
  }

  select {
    appearance: none;
    background-image: var(--highlight-gradient);
    color: var(--base00);
    border: 0;
  }

  option {
    color: initial;
  }

  :global(.reset) {
    color: var(--base05);
    text-decoration: none;
    line-height: normal;
    background-image: var(--button-gradient);
  }

  :global(.reset:hover), :global(.reset:active), :global(.reset:focus) {
    background-image: var(--button-hover-gradient);
  }
</style>
