import { Toaster } from "@plakar-ui/ui/components";
import { ComponentProps } from "react";
import { UNSTABLE_ToastQueue as ToastQueue } from "react-aria-components";

const DEFAULT_TIMEOUT = 5000;

type ToastContent = ComponentProps<typeof Toaster>;

export const toastQueue = new ToastQueue<ToastContent>();

export interface ToastOptions {
  content: ToastContent;
  sticky?: boolean;
  timeout?: number;
}

export function addToast({
  content,
  sticky = false,
  timeout = DEFAULT_TIMEOUT,
}: ToastOptions) {
  toastQueue.add(content, {
    timeout: sticky ? undefined : timeout,
  });
}
