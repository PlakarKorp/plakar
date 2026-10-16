import { Toaster } from "@plakar-ui/ui/components";
import clsx from "clsx";
import { useEffect, useState } from "react";
import {
  UNSTABLE_ToastRegion as ToastRegion,
  UNSTABLE_Toast as RACToast,
  UNSTABLE_ToastContent as ToastContent,
} from "react-aria-components";

import { toastQueue } from "./queue";

const getDefaultTitle = (variant: string = "default") =>
  ({
    default: "Information",
    info: "Information",
    brand: "Information",
    warning: "Warning",
    error: "Error",
    success: "Success",
  })[variant];

export function Toast() {
  const [entered, setEntered] = useState(false);

  useEffect(() => {
    requestAnimationFrame(() => setEntered(true));
  }, []);

  return (
    <ToastRegion
      queue={toastQueue}
      className="fixed top-4 left-1/2 z-100 -translate-x-1/2 max-md:w-full"
    >
      {({ toast }) => {
        const { content, key } = toast;
        const { variant, title } = content;
        const resolvedTitle = title ?? getDefaultTitle(variant);

        return (
          <RACToast
            toast={toast}
            style={{ viewTransitionName: key }}
            className={clsx("m-2", entered && "toast-enter")}
          >
            <ToastContent>
              <Toaster {...content} title={resolvedTitle} />
            </ToastContent>
          </RACToast>
        );
      }}
    </ToastRegion>
  );
}
