import clsx from "clsx";
import { useEffect, useState } from "react";
import { Label, ProgressBar as AriaProgressBar } from "react-aria-components";

type ProgressBarProps = {
  isLoading: boolean;
  className?: string;
};

export function ProgressBar({ isLoading, className }: ProgressBarProps) {
  const [visible, setVisible] = useState(false);
  const [progress, setProgress] = useState(0);

  useEffect(() => {
    let timer: ReturnType<typeof setTimeout> | null = null;
    let frame: ReturnType<typeof setInterval> | null = null;

    if (isLoading) {
      setVisible(true);
      setProgress(0);

      frame = setInterval(() => {
        setProgress((prev) => prev + ((100 - prev) / 100) * Math.random() * 5);
      }, 200);
    } else {
      setProgress(100);
      timer = setTimeout(() => {
        setVisible(false);
        setProgress(0);
      }, 300);
    }

    return () => {
      if (timer) clearTimeout(timer);
      if (frame) clearInterval(frame);
    };
  }, [isLoading]);

  if (!visible) return null;

  return (
    <AriaProgressBar
      value={progress}
      minValue={0}
      maxValue={100}
      className={clsx("h-1 w-full bg-transparent", className)}
    >
      <Label className="sr-only">Loading...</Label>
      <div
        className={clsx(
          "bg-accent h-full transition-all duration-200 ease-out"
        )}
        style={{ width: `${progress}%` }}
      />
    </AriaProgressBar>
  );
}
