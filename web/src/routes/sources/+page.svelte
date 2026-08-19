<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "$lib/api";
  import { ConfigDraft } from "$lib/draft.svelte";
  import Card from "$lib/ui/Card.svelte";
  import PageHeader from "$lib/ui/PageHeader.svelte";
  import Button from "$lib/ui/Button.svelte";
  import LinkButton from "$lib/ui/LinkButton.svelte";
  import ImportButton from "$lib/ui/ImportButton.svelte";
  import SaveBar from "$lib/ui/SaveBar.svelte";
  import Field from "$lib/ui/Field.svelte";
  import TextInput from "$lib/ui/TextInput.svelte";
  import NumberInput from "$lib/ui/NumberInput.svelte";
  import SelectInput from "$lib/ui/SelectInput.svelte";
  import EmptyState from "$lib/ui/EmptyState.svelte";

  const d = new ConfigDraft();
  onMount(() => d.init());

  function num(v: unknown): number | undefined {
    return typeof v === "number" ? v : typeof v === "string" && v !== "" ? Number(v) : undefined;
  }
  function str(v: unknown): string {
    return typeof v === "string" ? v : "";
  }
  function setOpt(obj: { options?: Record<string, unknown> }, name: string, value: unknown) {
    obj.options = { ...(obj.options ?? {}), [name]: value };
  }

  function addSource() {
    if (!d.cfg || !d.meta) return;
    const t = d.meta.sourceTypes[0];
    d.cfg.sources.push({
      id: `source-${d.cfg.sources.length + 1}`,
      type: t?.type ?? "midi",
      options: {},
    });
  }

  // All profiles for a source type: built-ins + user's custom boards.
  function profileChoices(sourceType: string): { value: string; label: string }[] {
    if (!d.meta || !d.cfg) return [];
    const builtin =
      d.meta.sourceTypes
        .find((t) => t.type === sourceType)
        ?.profiles?.map((p) => ({ value: p.id, label: `${p.id} — ${p.name}` })) ?? [];
    const custom = (d.cfg.profiles ?? [])
      .filter((p) => p.type === sourceType)
      .map((p) => ({ value: p.id, label: `${p.id} — ${p.name || "custom board"}` }));
    return [...builtin, ...custom];
  }
</script>

<PageHeader title="Sources">
  <LinkButton href={api.exportSectionURL("sources")} download="show-mapper-sources.yaml">
    ⭳ Export
  </LinkButton>
  <ImportButton onfile={(f) => d.importSection(f)} />
</PageHeader>

{#if !d.cfg || !d.meta}
  <p class="muted">Loading…</p>
{:else}
  <SaveBar onsave={() => d.save()} msg={d.msg} />

  {#if d.cfg.sources.length === 0}
    <EmptyState>
      <p>No sources yet. Sources are event producers: MIDI boards, virtual surfaces, …</p>
      <Button variant="primary" onclick={addSource}>+ Add your first source</Button>
      <p class="muted">
        For development without hardware/CGO: pick type <code>sim</code> —
        then play it on the <a href="/surface">Surface</a> tab.
      </p>
    </EmptyState>
  {/if}

  {#each d.cfg.sources as s, i (i)}
    <Card title="Source">
      {#snippet actions()}
        <Button variant="danger" onclick={() => d.cfg && d.cfg.sources.splice(i, 1)}>✕ remove</Button>
      {/snippet}
      <div class="rowline">
        <Field label="ID">
          <TextInput mono bind:value={s.id} />
        </Field>
        <Field label="Type">
          <SelectInput
            value={s.type}
            options={d.meta.sourceTypes.map((t) => t.type)}
            onchange={(e: Event) => {
              s.type = (e.currentTarget as HTMLSelectElement).value;
              s.profile = undefined;
            }}
          />
        </Field>
        {#if profileChoices(s.type).length > 0}
          <Field label="Board profile" grow>
            <SelectInput
              value={s.profile ?? ""}
              allowEmpty="(auto-detect from device name)"
              options={profileChoices(s.type)}
              onchange={(e: Event) => {
                const v = (e.currentTarget as HTMLSelectElement).value;
                s.profile = v || undefined;
              }}
            />
          </Field>
        {/if}
      </div>
      <div class="rowline">
        {#each d.meta.sourceTypes.find((t) => t.type === s.type)?.options ?? [] as f (f.name)}
          <Field label={f.label} hint={f.help} grow={f.type === "text"}>
            {#if f.type === "number"}
              <NumberInput
                value={num(s.options?.[f.name]) ?? num(f.default)}
                oninput={(e: Event) => setOpt(s, f.name, Number((e.currentTarget as HTMLInputElement).value))}
              />
            {:else}
              <TextInput
                value={str(s.options?.[f.name]) || str(f.default)}
                placeholder={f.help ?? ""}
                oninput={(e: Event) => setOpt(s, f.name, (e.currentTarget as HTMLInputElement).value)}
              />
            {/if}
          </Field>
        {:else}
          <span class="muted">No options for type {s.type}.</span>
        {/each}
      </div>
      {#if s.type === "midi"}
        <p class="muted">
          Unknown buttons show as <code>note:NN</code>/<code>cc:NN</code> in the Dashboard ticker —
          or create a <a href="/boards">custom board</a> / use
          <code>show-mapper midi monitor</code>.
        </p>
      {/if}
    </Card>
  {/each}
{/if}
