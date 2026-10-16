const ROWS = [0, 1, 2, 3];

export function Skeleton() {
  return (
    <div className="list skeleton" aria-hidden="true">
      <div className="day-head">
        <span className="skeleton-date">
          <span className="skeleton-bar skeleton-day" />
          <span className="skeleton-bar skeleton-month" />
        </span>
        <span className="skeleton-bar skeleton-sum" />
      </div>
      {ROWS.map((i) => (
        <div className="expense" key={i}>
          <span className="person">
            <span className="avatar skeleton-avatar" />
            <span className="skeleton-bar sk-name" />
          </span>
          <span className="skeleton-bar sk-cat" />
          <span className="skeleton-bar sk-comment" />
          <span className="skeleton-bar skeleton-amount" />
          <span />
        </div>
      ))}
    </div>
  );
}
