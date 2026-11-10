export function PageHeading({ eyebrow, title, description, actions }: {
  eyebrow: string;
  title: React.ReactNode;
  description: string;
  actions?: React.ReactNode;
}) {
  return (
    <header className="cw-page-heading">
      <div className="min-w-0">
        <p className="cw-eyebrow mb-2 uppercase">{eyebrow}</p>
        <h1 className="cw-page-title">{title}</h1>
        <p className="cw-page-description">{description}</p>
      </div>
      {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
    </header>
  );
}
