<script lang="ts">
  import { onMount } from "svelte";
  import { api, ApiError } from "$lib/api";
  import { ConfigDraft } from "$lib/draft.svelte";
  import type { NICInfo } from "$lib/types";
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
  import Msg from "$lib/ui/Msg.svelte";

  const d = new ConfigDraft();
  onMount(() => d.init());

  let ifaces = $state<NICInfo[] | null>(null);
  let ifaceMsg = $state<{ text: string; ok: boolean } | null>(null);

  async function toggleIfaces() {
    if (ifaces) {
      ifaces = null;
      return;
    }
    ifaceMsg = null;
    try {
      ifaces = (await api.interfaces()).interfaces;
    } catch (e) {
      ifaceMsg = { text: e instanceof ApiError ? e.errors.join("; ") : String(e), ok: false };
    }
  }

  function num(v: unknown): number | undefined {
    return typeof v === "number" ? v : typeof v === "string" && v !== "" ? Number(v) : undefined;
  }
  function str(v: unknown): string {
    return typeof v === "string" ? v : "";
  }
  function setOpt(obj: { options?: Record<string, unknown> }, name: string, value: unknown) {
    obj.options = { ...(obj.options ?? {}), [name]: value };
  }

  function addTarget() {
    if (!d.cfg || !d.meta) return;
    d.cfg.targets.push({
      id: `target-${d.cfg.targets.length + 1}`,
      type: d.meta.targetTypes[0]?.type ?? "osc",
      options: {},
    });
  }
</script>

<PageHeader title="Targets">
  <LinkButton href={api.exportSectionURL("targets")} download="show-mapper-targets.yaml">
    ⭳ Export
  </LinkButton>
  <ImportButton onfile={(f) => d.importSection(f)} />
</PageHeader>

{#if !d.cfg || !d.meta}
  <p class="muted">Loading…</p>
{:else}
  <SaveBar onsave={() => d.save()} msg={d.msg} />

  {#if d.cfg.targets.length === 0}
    <EmptyState>
      <p>No targets yet — targets receive actions (e.g. an OSC console).</p>
      <Button variant="primary" onclick={addTarget}>+ Add your first target</Button>
    </EmptyState>
  {/if}

  {#each d.cfg.targets as t, i (i)}
    <Card title="Target">
      {#snippet actions()}
        <Button variant="danger" onclick={() => d.cfg && d.cfg.targets.splice(i, 1)}>✕ remove</Button>
      {/snippet}
      <div class="rowline">
        <Field label="ID">
          <TextInput mono bind:value={t.id} />
        </Field>
        <Field label="Type">
          <SelectInput
            value={t.type}
            options={d.meta.targetTypes.map((ty) => ty.type)}
            onchange={(e: Event) => {
              t.type = (e.currentTarget as HTMLSelectElement).value;
            }}
          />
        </Field>
        <div class="spread"></div>
        <Field label="Network interfaces">
          <Button variant="ghost" onclick={toggleIfaces}>
            {ifaces ? "Hide" : "Show"} list
          </Button>
        </Field>
      </div>
      <div class="rowline">
        {#each d.meta.targetTypes.find((ty) => ty.type === t.type)?.options ?? [] as f (f.name)}
          <Field label={f.label} hint={f.help} grow={f.type === "text"}>
            {#if f.type === "number"}
              <NumberInput
                value={num(t.options?.[f.name]) ?? num(f.default)}
                oninput={(e: Event) => setOpt(t, f.name, Number((e.currentTarget as HTMLInputElement).value))}
              />
            {:else}
              <TextInput
                value={str(t.options?.[f.name]) || str(f.default)}
                placeholder={f.help ?? ""}
                oninput={(e: Event) => setOpt(t, f.name, (e.currentTarget as HTMLInputElement).value)}
              />
            {/if}
          </Field>
        {:else}
          <span class="muted">No options for type {t.type}.</span>
        {/each}
      </div>
    </Card>
  {/each}

  {#if ifaces}
    <Card title="Network interfaces">
      <table>
        <thead>
          <tr><th>Name</th><th>IPv4</th><th>Up</th><th>Multicast</th><th>Notes</th></tr>
        </thead>
        <tbody>
          {#each ifaces as n (n.name)}
            <tr>
              <td class="mono">{n.name}</td>
              <td class="mono">{n.ipv4.length ? n.ipv4.join(", ") : "—"}</td>
              <td>{n.up ? "✓" : ""}</td>
              <td>{n.multicast ? "✓" : ""}</td>
              <td class="muted">{n.loopback ? "loopback" : ""}</td>
            </tr>
          {/each}
        </tbody>
      </table>
      <p class="muted">
        Use an IPv4 address as a target's <code>localAddress</code> to pin its socket to that NIC.
        Multiple target instances can bind different NICs simultaneously.
      </p>
    </Card>
  {/if}
  <Msg msg={ifaceMsg} />
{/if}
