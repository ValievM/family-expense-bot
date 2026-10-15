/** Свойства баннера ошибки. */
interface ErrorBannerProps {
  message: string;
  onRetry?: () => void;
}

/**
 * Принимает: текст ошибки и необязательный обработчик повтора.
 * Делает: показывает сообщение с role="alert" и кнопкой "Повторить".
 * Возвращает: React-элемент баннера.
 */
export function ErrorBanner({ message, onRetry }: ErrorBannerProps) {
  return (
    <div className="banner banner--error" role="alert">
      <span>{message}</span>
      {onRetry && (
        <button type="button" className="btn btn--small" onClick={onRetry}>
          Повторить
        </button>
      )}
    </div>
  );
}
