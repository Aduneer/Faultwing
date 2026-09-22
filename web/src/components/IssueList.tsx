import type { Issue, IssueStatus } from "../api/flytrap";
import { bugAssetForIssue } from "./Bug";

const statuses: IssueStatus[] = ["open", "resolved", "ignored"];

function relativeTime(value: string) {
  const elapsedMinutes = Math.max(0, Math.round((Date.now() - Date.parse(value)) / 60_000));
  if (elapsedMinutes < 60) return `${Math.max(1, elapsedMinutes)} min ago`;
  const hours = Math.round(elapsedMinutes / 60);
  if (hours < 24) return `${hours} hr ago`;
  const days = Math.round(hours / 24);
  return `${days} day${days === 1 ? "" : "s"} ago`;
}

type IssueListProps = {
  issues: Issue[];
  selectedId: number | null;
  filter: IssueStatus;
  onFilter: (status: IssueStatus) => void;
  onSelect: (id: number) => void;
  hasMore: boolean;
  loadingMore: boolean;
  onLoadMore: () => void;
};

export function IssueList({
  issues,
  selectedId,
  filter,
  onFilter,
  onSelect,
  hasMore,
  loadingMore,
  onLoadMore,
}: IssueListProps) {
  return (
    <section className="issue-list" aria-label="Issues">
      <div className="status-filter" aria-label="Issue status filter">
        {statuses.map((status) => (
          <button
            key={status}
            type="button"
            className={`status-filter__button ${filter === status ? "is-active" : ""}`}
            aria-pressed={filter === status}
            onClick={() => onFilter(status)}
          >
            {status}
          </button>
        ))}
      </div>

      <div className="issue-list__head" aria-hidden="true">
        <span>Issue</span>
        <span>Events</span>
        <span>Last seen</span>
        <span />
      </div>

      {issues.length === 0 ? (
        <div className="issue-list__empty">
          <strong>No {filter} issues</strong>
          <span>This view will update when matching issues arrive.</span>
        </div>
      ) : (
        <ul>
          {issues.map((issue) => (
            <li key={issue.id}>
              <button
                type="button"
                className={`issue-list__item ${selectedId === issue.id ? "issue-list__item--selected" : ""}`}
                onClick={() => onSelect(issue.id)}
                aria-current={selectedId === issue.id ? "true" : undefined}
              >
                <span className="issue-list__identity">
                  <img src={bugAssetForIssue(issue)} alt="" aria-hidden="true" />
                  <span>
                    <strong>{issue.exception_type}</strong>
                    <small>{issue.message}</small>
                  </span>
                </span>
                <span className="issue-list__events">{issue.event_count}</span>
                <time dateTime={issue.last_seen}>{relativeTime(issue.last_seen)}</time>
                <span className="issue-list__chevron" aria-hidden="true">›</span>
              </button>
            </li>
          ))}
        </ul>
      )}

      {hasMore && (
        <button type="button" className="button issue-list__more" disabled={loadingMore} onClick={onLoadMore}>
          {loadingMore ? "Loading…" : "Load more"}
        </button>
      )}
    </section>
  );
}
