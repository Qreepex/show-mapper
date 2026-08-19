<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "$lib/api";
  import { ConfigDraft } from "$lib/draft.svelte";
  import type { ControlKind, ProfileConfig } from "$lib/types";
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
  import CheckboxInput from "$lib/ui/CheckboxInput.svelte";
  import EmptyState from "$lib/ui/EmptyState.svelte";

  const d = new ConfigDraft();
  onMount(() => d.init());

  function newProfile(): ProfileConfig {
    return {
      id: `board-${(d.cfg?.profiles?.length ?? 0) + 1}`,
      type: "midi",
      name: "",
      match: [],
      led: { style: "onOff" },
      controls: [],
    };
  }

  function addControl(p: ProfileConfig) {
    p.controls.push({ id: `ctrl-${p.controls.length + 1}`, kind: "button", label: "", hasLED: false });
  }

  function setNum(c: { note?: number; cc?: number }, field: "note" | "cc", raw: string) {
    c[field] = raw === "" ? undefined : Math.trunc(Number(raw));
  }
  function csvToList(v: string): string[] {
    return v.split(",").map((s) => s.trim()).filter(Boolean);
  }
</script>

<PageHeader title="Boards (custom profiles)">
  <LinkButton href={api.exportSectionURL("profiles")} download="show-mapper-profiles.yaml">
    ⭳ Export boards
  </LinkButton>
  <ImportButton onfile={(f) => d.importSection(f)} />
</PageHeader>

{#if !d.cfg || !d.meta}
  <p class="muted">Loading…</p>
{:else}
  <SaveBar onsave={() => d.save()} msg={d.msg} />

  <Card title="Built-in boards">
    <table>
      <thead>
        <tr><th>ID</th><th>Name</th><th>Controls</th><th>LED</th></tr>
      </thead>
      <tbody>
        {#each d.meta.sourceTypes.flatMap((t) => t.profiles ?? []) as p (p.id)}
          <tr>
            <td class="mono">{p.id}</td>
            <td>{p.name}</td>
            <td class="muted">{p.controls.length}</td>
            <td class="muted">{p.led}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  </Card>

  {#if (d.cfg.profiles ?? []).length === 0}
    <EmptyState>
      <p>
        Any MIDI board can be described here — discover its numbers on the Dashboard ticker
        (<code>note:NN</code>/<code>cc:NN</code>) or via <code>show-mapper midi monitor</code>.
      </p>
      <Button variant="primary" onclick={() => d.cfg && (d.cfg.profiles = [...(d.cfg.profiles ?? []), newProfile()])}>
        + Describe a board
      </Button>
    </EmptyState>
  {/if}

  {#each d.cfg.profiles ?? [] as p, pi (pi)}
    <Card title={p.name || p.id}>
      {#snippet actions()}
        <Button variant="danger" onclick={() => d.cfg && d.cfg.profiles && d.cfg.profiles.splice(pi, 1)}>
          ✕ remove
        </Button>
      {/snippet}
      <div class="rowline">
        <Field label="ID">
          <TextInput mono bind:value={p.id} />
        </Field>
        <Field label="Name" grow>
          <TextInput bind:value={p.name} />
        </Field>
        <Field label="LED style">
          <SelectInput value={p.led?.style ?? "none"} options={d.meta.ledStyles}
            onchange={(e: Event) => { p.led = { ...p.led, style: (e.currentTarget as HTMLSelectElement).value }; }} />
        </Field>
      </div>
      <div class="rowline">
        <Field label="Match (comma-separated port-name substrings)" grow>
          <TextInput value={(p.match ?? []).join(", ")}
            oninput={(e: Event) => { p.match = csvToList((e.currentTarget as HTMLInputElement).value); }} />
        </Field>
      </div>

      <table>
        <thead>
          <tr><th>ID</th><th>Label</th><th>Kind</th><th>Note (0–127)</th><th>CC (0–127)</th><th>LED</th><th></th></tr>
        </thead>
        <tbody>
          {#each p.controls as c, ci (ci)}
            <tr>
              <td><TextInput mono bind:value={c.id} /></td>
              <td><TextInput bind:value={c.label} /></td>
              <td>
                <SelectInput value={c.kind} options={d.meta.controlKinds}
                  onchange={(e: Event) => { c.kind = (e.currentTarget as HTMLSelectElement).value as ControlKind; }} />
              </td>
              <td>
                <NumberInput min={0} max={127} value={c.note}
                  oninput={(e: Event) => setNum(c, "note", (e.currentTarget as HTMLInputElement).value)} />
              </td>
              <td>
                <NumberInput min={0} max={127} value={c.cc}
                  oninput={(e: Event) => setNum(c, "cc", (e.currentTarget as HTMLInputElement).value)} />
              </td>
              <td><CheckboxInput bind:checked={c.hasLED} /></td>
              <td><Button variant="danger" onclick={() => p.controls.splice(ci, 1)}>✕</Button></td>
            </tr>
          {/each}
        </tbody>
      </table>
      <div class="row mt">
        <Button onclick={() => addControl(p)}>+ Add control</Button>
      </div>
    </Card>
  {/each}

  <Button variant="primary" onclick={() => d.cfg && (d.cfg.profiles = [...(d.cfg.profiles ?? []), newProfile()])}>
    + Add board
  </Button>
{/if}
