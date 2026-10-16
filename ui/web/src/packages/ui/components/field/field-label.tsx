import clsx from "clsx";
import { ElementType, PropsWithChildren, ReactNode } from "react";
import { ButtonContext, Label } from "react-aria-components";

import { Typography } from "../typography/typography";

type FieldLabelProps = PropsWithChildren & {
  as?: ElementType;
  className?: string;
  isRequired?: boolean;
  beforeAction?: ReactNode;
  action?: ReactNode;
  id?: string;
};

export function FieldLabel({
  children,
  as,
  className,
  isRequired = false,
  beforeAction,
  action,
  id,
}: FieldLabelProps) {
  const label = (
    <>
      <Typography
        as={as ?? ((props) => <Label {...props} />)}
        id={id}
        size="sm"
        weight="medium"
        className={clsx(
          "text-secondary group-data-disabled:text-tertiary self-start",
          className
        )}
      >
        {children}
      </Typography>
      {isRequired && (
        <Typography as="span" className="text-error" aria-hidden="true">
          *
        </Typography>
      )}
    </>
  );

  if (!beforeAction && !action) {
    return <div>{label}</div>;
  }

  return (
    <div className="flex h-6 items-center gap-2">
      <ButtonContext.Provider value={null}>
        {beforeAction}
        <div>{label}</div>
        {action}
      </ButtonContext.Provider>
    </div>
  );
}
