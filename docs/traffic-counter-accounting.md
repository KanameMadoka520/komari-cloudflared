# Traffic counter accounting

The server accepts one traffic decision per observation and persists its state
with the lifetime and configured billing totals. History and notifications use
that same decision. Raw agent counters remain available as raw measurements.

## Legacy agents

A decrease in an aggregate interface counter does not prove a reset. Interface
selection can change and temporary virtual interfaces can disappear. The former
64 GiB heuristic could add a historical total as fresh traffic. Returning zero
for only the decreasing sample is also insufficient: counting its subsequent
recovery would add old bytes again.

For reports without explicit counter metadata, each direction retains its
persisted high-water mark. Lower values add zero, and later growth counts only
above that mark. The affected intervals and retained account totals are flagged
incomplete. This is a conservative lower bound: after a genuine reset, usage may
remain understated until the old mark is exceeded. Server-side calibration of a
billing total does not reset the independent counter state or lifetime total.
The server cannot recover exact traffic, processes, or destinations from old
aggregate observations.

## Optional agent protocol

`network.counterEpoch` is an optional unsigned, monotonically increasing,
durably persisted generation. Both counters must start at zero whenever this
generation increases. It must identify the counter domain, rather than the
process or boot: a boot ID or timestamp attached to still-existing OS totals
would falsely count those totals on every restart. Agents must retain a stable
domain across restarts or create genuinely zero-based software counters.

On first enrollment, the server establishes a baseline without adding existing
bytes. Later increases in an already enrolled epoch confirm a reset and allow
counting the new zero-based totals. Older generations, including missing
metadata after enrollment, cannot roll the accepted state backwards.

`network.counterBits` may be 0, 32 or 64. Explicit 32-bit mode requires a positive
epoch and individual counters below 2^32. It is only valid for a single modulo
counter per direction, not sums of independently wrapping interfaces. A wrap
from the upper quarter to the lower quarter counts the full modular tail. Other
decreases are uncertain. Sampling cannot prove or recover multiple unseen wraps;
agents should prefer durable 64-bit software totals.

The bundled server patch does not upgrade remote agent binaries. Existing agents
continue to use conservative legacy accounting until a compatible agent is
implemented and deployed. This protocol is a contract, not a claim that existing
agent boot identifiers satisfy it.

## Durability, retention and incomplete measurements

Main-DB updates persist counter state and account totals atomically. Server-only
accepted deltas cannot be supplied through agent JSON. Accepted lifetime totals
are also recorded in the metric store. After a queue overflow or crash, the next
observation bridges missing queued deltas from these accepted totals and marks
the recovered interval incomplete because exact timestamps were lost. Missing
history at first enrollment cannot be recovered from an absent baseline.

Gaps longer than 15 minutes do not assign offline traffic to one history bucket;
they contribute zero to that bucket and an incomplete marker. Lifetime and
configured billing totals can still retain confirmed counter growth. Minute,
hour, and day rollups sum directional deltas and quality flags. An explicitly
stored zero remains zero; notifications fall back to raw differences only when
a directional delta is actually absent.

Raw counter/high-water and accepted-total metrics are retained using the store's
normal policy. Account state remains in the main DB independently of history
retention. Restoration from floating-point history has the same byte precision
limit as the existing metric store; counters above 2^53 cannot retain individual
byte precision there. Historical inflated deltas are not rewritten by this patch.

The sticky account-level incomplete flag means a retained total contains at least
one uncertain interval. A report's period-level marker refers to uncertainty in
that specific period. These flags should not be cleared merely by rebooting or
calibrating one billing value: lifetime uncertainty can remain.
