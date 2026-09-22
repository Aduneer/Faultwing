import type { Issue } from "../api/flytrap";
import Bug from "./Bug";

type TerrariumProps = {
  issues: Issue[];
  selectedId: number | null;
  onSelect: (id: number) => void;
};

const positions = [
  { x: 34, y: 52 },
  { x: 60, y: 62 },
  { x: 76, y: 45 },
  { x: 49, y: 39 },
  { x: 23, y: 59 },
  { x: 73, y: 66 },
  { x: 44, y: 68 },
  { x: 84, y: 57 },
  { x: 24, y: 42 },
  { x: 66, y: 37 },
  { x: 50, y: 55 },
  { x: 30, y: 67 },
];

function plot(index: number, issue: Issue) {
  const base = positions[index % positions.length];
  const layer = Math.floor(index / positions.length);
  const jitterX = ((Math.abs(issue.id * 17) % 7) - 3) * 0.45;
  const jitterY = ((Math.abs(issue.id * 29) % 5) - 2) * 0.35;

  return {
    x: Math.max(14, Math.min(86, base.x + jitterX + layer)),
    y: Math.max(30, Math.min(70, base.y + jitterY + layer)),
  };
}

export function Terrarium({ issues, selectedId, onSelect }: TerrariumProps) {
  const openIssues = issues.filter((issue) => issue.status === "open");
  const displayed = openIssues.slice(0, positions.length);

  return (
    <div className="terrarium" aria-label="Issue terrarium">
      <div className="terrarium__stage">
        <div className="terrarium__ambient" aria-hidden="true" />
        <img
          className="terrarium__image"
          src="/assets/terrarium.webp"
          alt=""
          aria-hidden="true"
          draggable={false}
        />

        {displayed.map((issue, index) => (
          <Bug
            key={issue.id}
            issue={issue}
            selected={issue.id === selectedId}
            onSelect={onSelect}
            position={plot(index, issue)}
          />
        ))}

        {!displayed.length && (
          <div className="terrarium__empty">
            <strong>The habitat is peaceful.</strong>
            <span>No open issues in this view.</span>
          </div>
        )}

        {openIssues.length > displayed.length && (
          <span className="terrarium__more">+{openIssues.length - displayed.length} more</span>
        )}
      </div>
    </div>
  );
}

export default Terrarium;
