interface ErrorTextProps {
  message?: string | null;
}

export function ErrorText({ message }: ErrorTextProps) {
  if (!message) {
    return null;
  }

  return <p className="error-text">{message}</p>;
}
