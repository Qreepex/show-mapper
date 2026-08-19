# target module: `osc` - generic OSC over UDP

Connector type id: **`osc`**. Sends Open Sound Control messages to any
OSC-capable receiver (lighting consoles, media servers, audio software).
Deliberately generic - console-specific sugar lives in helper modules
(e.g. [`internal/helpers/gma3`](../helpers/gma3/README.md) for grandMA3).

## Options

| option | type | required | default | notes |
| --- | --- | --- | --- | --- |
| `host` | text | yes | — | receiver IP (unicast; broadcast also works) |
| `port` | number | yes | 8000 | receiver UDP port |
| `prefix` | text | no | "" | prepended to every address, no slashes.<br>`prefix: lights` + `/cmd` → `/lights/cmd` |
| `localAddress` | text | no | "" | local IPv4 (NIC) the outgoing socket binds to, for multi-homed machines. List NICs at `GET /api/system/interfaces` or Settings → “Show network interfaces”. Several instances may each bind a different NIC at the same time. |

## How actions map to OSC

show-mapper `ActionConfig` → one OSC message (bundles intentionally not used -
receivers like grandMA3 reject them):

| action type | address | payload |
| --- | --- | --- |
| `command` | `address` | one **string** (e.g. receiver-side command syntax) |
| `value` | `address` | `pressValue` / `releaseValue` as int32 (default) or float32 |
| `fader` | `address` | source control value (0..1) scaled into `range`, int32/float32 |

Library: `github.com/hypebeast/go-osc` (pure Go, MIT).

## Adding this module to a build

Registered in `cmd/show-mapper/main.go` via blank import
(`_ "internal/targets/osc"`). Removing the import removes the module from the
binary - show-mapper core runs fine without it.
