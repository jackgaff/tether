interface JsonViewProps {
  value: unknown;
  emptyLabel?: string;
}

export function JsonView({ value, emptyLabel = "No data." }: JsonViewProps) {
  if (value === null || value === undefined) {
    return <pre>{emptyLabel}</pre>;
  }

  return <pre>{JSON.stringify(value, null, 2)}</pre>;
}
