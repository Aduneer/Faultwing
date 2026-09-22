import type { CSSProperties } from "react";
import type { Issue } from "../api/faultwing";

const bugAssets = [
  "fruitfly",
  "ladybug",
  "butterfly",
  "dragonfly",
  "moth",
  "stinkbug",
  "weevil",
  "firefly",
  "aphid",
  "caterpillar",
  "pillbug",
] as const;

type BugProps = {
  issue: Issue;
  selected: boolean;
  onSelect: (id: number) => void;
  position: { x: number; y: number };
};

export function bugAssetForIssue(issue: Pick<Issue, "id" | "fingerprint">) {
  const fingerprintSeed = [...issue.fingerprint].reduce((sum, character) => sum + character.charCodeAt(0), 0);
  const index = Math.abs(issue.id * 17 + fingerprintSeed) % bugAssets.length;
  return `/assets/${bugAssets[index]}.webp`;
}

export default function Bug({ issue, selected, onSelect, position }: BugProps) {
  const scale = Math.min(1.35, Math.max(0.72, 0.72 + Math.log10(Math.max(1, issue.event_count)) * 0.24));
  const style = {
    "--bug-x": `${position.x}%`,
    "--bug-y": `${position.y}%`,
    "--bug-scale": scale,
    "--bug-delay": `${(Math.abs(issue.id) * 0.37) % 3.7}s`,
  } as CSSProperties;

  return (
    <button
      type="button"
      className={`bug ${selected ? "bug--selected" : ""}`}
      style={style}
      onClick={() => onSelect(issue.id)}
      aria-label={`Select ${issue.exception_type}: ${issue.message}. ${issue.event_count} events.`}
      aria-pressed={selected}
    >
      {selected && <span className="bug__marker" aria-hidden="true">#{issue.id}</span>}
      <span className="bug__stem" aria-hidden="true" />
      <span className="bug__halo" aria-hidden="true" />
      <img src={bugAssetForIssue(issue)} alt="" draggable={false} />
    </button>
  );
}
