import { Icon } from './Icon';

interface ErrorBannerProps {
  message: string;
  details?: string;
  onRetry?: () => void;
  retryLabel?: string;
}

export function ErrorBanner({ message, details, onRetry, retryLabel = 'Повторить' }: ErrorBannerProps) {
  return (
    <div className="banner" role="alert">
      <Icon name="alert" />
      <div className="banner-text">
        <p>{message}</p>
        {details && (
          <details>
            <summary>Технические подробности</summary>
            <code>{details}</code>
          </details>
        )}
      </div>
      {onRetry && (
        <button type="button" onClick={onRetry}>
          {retryLabel}
        </button>
      )}
    </div>
  );
}
