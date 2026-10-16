import { Icon, Typography } from "..";

export function DemoBanner() {
  return (
    <Typography
      as="div"
      size="sm"
      className="bg-accent-tertiary text-accent grid grid-cols-[auto_1fr] gap-x-2 rounded-md px-2 py-1"
    >
      <div className="row-start-1 flex items-center">
        <Icon type="info-circle" size="sm" />
      </div>
      <p className="row-start-1">
        <strong>Demo Mode</strong>
      </p>
      <Typography size="xs" className="text-secondary col-start-2 row-start-2">
        You're signed in as a demo user.
      </Typography>
    </Typography>
  );
}
