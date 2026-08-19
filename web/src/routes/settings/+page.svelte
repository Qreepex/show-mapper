<script lang="ts">
  import { api } from "$lib/api";
  import { ConfigDraft } from "$lib/draft.svelte";
  import Card from "$lib/ui/Card.svelte";
  import CheckboxInput from "$lib/ui/CheckboxInput.svelte";
  import Field from "$lib/ui/Field.svelte";
  import ImportButton from "$lib/ui/ImportButton.svelte";
  import LinkButton from "$lib/ui/LinkButton.svelte";
  import PageHeader from "$lib/ui/PageHeader.svelte";
  import SaveBar from "$lib/ui/SaveBar.svelte";
  import TextInput from "$lib/ui/TextInput.svelte";
  import { live } from "$lib/ws.svelte";
  import { onMount } from "svelte";

  // Application-level settings only. Sources / Targets / Boards have their
  // own tabs (top nav).
  const d = new ConfigDraft();
  onMount(() => d.init());
</script>

<PageHeader title="Settings" />

{#if !d.cfg}
  <p class="muted">Loading…</p>
{:else}
  <SaveBar onsave={() => d.save()} msg={d.msg} />

  <Card title="Web UI / HTTP">
    <Field
      label="Listen address"
      hint="localhost-only by default; use 0.0.0.0:8484 to expose on a trusted show network (no auth yet)."
    >
      <TextInput mono bind:value={d.cfg.http.listen} />
    </Field>
  </Card>

  <Card title="Software updates">
    <div class="rowline">
      <Field
        label="GitHub repo for releases"
        hint="'owner/name' — empty disables the feature"
      >
        <TextInput
          mono
          value={d.cfg.updates?.repo ?? ""}
          placeholder="Qreepex/show-mapper"
          oninput={(e: Event) => {
            if (!d.cfg) return;
            d.cfg.updates = {
              repo: (e.currentTarget as HTMLInputElement).value,
              autoCheck: d.cfg.updates?.autoCheck ?? false,
            };
          }}
        />
      </Field>
      <Field label="Startup check">
        <CheckboxInput
          label="Check automatically on startup"
          checked={d.cfg.updates?.autoCheck ?? false}
          onchange={(e: Event) => {
            if (!d.cfg) return;
            d.cfg.updates = {
              repo: d.cfg.updates?.repo ?? "",
              autoCheck: (e.currentTarget as HTMLInputElement).checked,
            };
          }}
        />
      </Field>
    </div>
  </Card>

  <Card title="Config file">
    <div class="row">
      <LinkButton href={api.exportConfigURL} download="show-mapper.yaml"
        >⭳ Download full config (YAML)</LinkButton
      >
      <ImportButton
        label="⭱ Import full config (YAML)…"
        onfile={(f) => d.importConfig(f)}
      />
      <span class="muted"
        >Move configs between machines; hand edits also hot-reload from disk.</span
      >
    </div>
  </Card>

  <Card title="About">
    <div class="row">
      <span class="mono muted"
        >version {live.version} · commit {live.commit}</span
      >
    </div>
    <p class="muted">
      Sources, targets and custom boards moved to their own tabs in the top
      navigation. Architecture &amp; module docs live in the repo's <code
        >docs/</code
      > and module READMEs.
    </p>
  </Card>
{/if}
