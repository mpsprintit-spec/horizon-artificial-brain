export type BridgeStatus = "disconnected" | "connecting" | "connected" | "stale" | "resyncing" | "error";

export interface BrainSnapshot {
  type: string;
  brain_identity: string;
  state_revision: number;
  captured_at: string;
  canonical_state_hash: string;
  counts: Record<string, number>;
  neural_units: unknown[];
  populations: unknown[];
  synapses: unknown[];
  temporal_patterns: unknown[];
  experience_traces: unknown[];
  brain_state: Record<string, unknown>;
}

export interface TelemetryEvent {
  type: string;
  brain_identity: string;
  state_revision: number;
  timestamp: string;
  event_hash?: string;
  canonical_state_hash?: string;
  [key: string]: unknown;
}

export class HorizonStore {
  private snapshotValue: BrainSnapshot | null = null;
  private eventsValue: TelemetryEvent[] = [];
  private statusValue: BridgeStatus = "disconnected";
  private listeners = new Set<() => void>();
  private gapValue: { expected: number; received: number } | null = null;

  get snapshot(): BrainSnapshot | null { return this.snapshotValue; }
  get events(): readonly TelemetryEvent[] { return this.eventsValue; }
  get status(): BridgeStatus { return this.statusValue; }
  get gap(): { expected: number; received: number } | null { return this.gapValue; }
  get revision(): number { return this.snapshotValue?.state_revision ?? 0; }

  subscribe(listener: () => void): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  setStatus(status: BridgeStatus): void { this.statusValue = status; this.emit(); }

  setSnapshot(snapshot: BrainSnapshot): void {
    if (snapshot.type !== "brain.snapshot") throw new Error("invalid brain snapshot type");
    if (!Number.isFinite(snapshot.state_revision)) throw new Error("invalid snapshot revision");
    this.snapshotValue = snapshot;
    this.gapValue = null;
    this.emit();
  }

  applyEvent(event: TelemetryEvent): "applied" | "duplicate" | "gap" | "stale" {
    if (event.brain_identity && this.snapshotValue && event.brain_identity !== this.snapshotValue.brain_identity) {
      throw new Error("telemetry brain identity mismatch");
    }
    const revision = Number(event.state_revision);
    if (!Number.isFinite(revision)) throw new Error("telemetry event has invalid revision");
    if (revision <= this.revision) return "duplicate";
    if (revision > this.revision + 1) {
      this.gapValue = { expected: this.revision + 1, received: revision };
      this.statusValue = "resyncing";
      this.emit();
      return "gap";
    }
    this.eventsValue = [...this.eventsValue.slice(-499), event];
    if (event.canonical_state_hash && this.snapshotValue && event.state_revision === this.snapshotValue.state_revision && event.canonical_state_hash !== this.snapshotValue.canonical_state_hash) {
      this.statusValue = "error";
      this.emit();
      throw new Error("canonical state hash mismatch");
    }
    this.emit();
    return "applied";
  }

  replaceEvents(events: TelemetryEvent[]): void { this.eventsValue = events.slice(-500); this.emit(); }
  clearForReconnect(): void { this.statusValue = "connecting"; this.emit(); }
  private emit(): void { for (const listener of this.listeners) listener(); }
}
