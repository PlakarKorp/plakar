import clsx from "clsx";
import { useState } from "react";
import { ProgressBar } from "react-aria-components";

import { Icon, Typography } from "..";

export type Status = "complete" | "current" | "upcoming";

type StepperProps<T extends readonly string[]> = {
  steps: T;
  children: (props: {
    currentStep: T[number];
    setProgress: (value: number) => void;
    goToPreviousStep: () => void;
    goToNextStep: () => void;
  }) => React.ReactNode;
};

export function Stepper<T extends readonly string[]>({
  steps,
  children,
}: StepperProps<T>) {
  const [progress, setProgress] = useState(0);
  const [currentStep, setCurrentStep] = useState<T[number]>(steps[0]);

  const currentIndex = steps.indexOf(currentStep);
  if (currentIndex === -1) {
    throw new Error(
      `Current step "${currentStep}" not found in steps: [${steps.join(", ")}]`
    );
  }

  const goToPreviousStep = () => {
    if (currentIndex > 0) {
      setProgress(0);
      setCurrentStep(steps[currentIndex - 1]);
    }
  };

  const goToNextStep = () => {
    if (currentIndex < steps.length - 1) {
      setProgress(0);
      setCurrentStep(steps[currentIndex + 1]);
    }
  };

  return (
    <div className="flex w-full flex-col gap-8">
      <style>{`
        @keyframes plk-check-pop {
          0%   { transform: scale(0) rotate(-45deg); opacity: 0; }
          60%  { transform: scale(1.33) rotate(0deg);  opacity: 1; }
          100% { transform: scale(1)    rotate(0deg);  opacity: 1; }
        }
      `}</style>

      <ol
        className="border-secondary grid w-full items-start gap-2 border-b pb-8"
        style={{
          gridTemplateColumns: `repeat(${Math.max(steps.length - 1, 0)}, minmax(0,1fr)) auto`,
        }}
      >
        {steps?.map((step, index) => (
          <Step
            key={step}
            index={index + 1}
            title={step}
            status={
              index < currentIndex ||
              (index === currentIndex && progress === 100)
                ? "complete"
                : index === currentIndex
                  ? "current"
                  : "upcoming"
            }
            progress={progress}
            isLast={index === steps.length - 1}
          />
        ))}
      </ol>

      {children({
        currentStep,
        setProgress,
        goToPreviousStep,
        goToNextStep,
      })}
    </div>
  );
}

type StepProps = {
  status?: Status;
  progress?: number;
  index: number;
  title?: string;
  isLast?: boolean;
};

function Step({
  status = "upcoming",
  progress = 0,
  index,
  title,
  isLast = false,
}: StepProps) {
  const percentage = Math.max(0, Math.min(100, progress));

  return (
    <li className="flex w-full flex-col gap-2">
      <div className="flex w-full items-center gap-2">
        <div
          className={clsx(
            "transition-p text-on-accent shrink-0 rounded-full border-2 duration-300 ease-out",
            {
              complete: "border-background-success bg-success",
              current: "border-accent bg-accent border-2 bg-clip-content p-1",
              upcoming: "border-background-accent-tertiary bg-accent-tertiary",
            }[status],
            "flex h-8 w-8 items-center justify-center"
          )}
        >
          {status === "complete" && (
            <Icon
              type="check"
              size="md"
              className="will-change-transform"
              style={{
                animation: "plk-check-pop 380ms cubic-bezier(.2,.9,.25,1) both",
                transformOrigin: "center",
              }}
            />
          )}
        </div>
        {!isLast && (
          <ProgressBar
            aria-label={`Step ${index} progress`}
            className="bg-accent-tertiary h-1 w-full rounded-full"
            minValue={status === "current" ? 0 : undefined}
            maxValue={status === "current" ? 100 : undefined}
            value={status === "current" ? percentage : undefined}
          >
            <div
              className={clsx(
                "h-full rounded-full transition-all duration-300 ease-out",
                status === "complete" ? "bg-success" : "bg-accent"
              )}
              style={{
                width: `${{ complete: 100, current: percentage, upcoming: 0 }[status]}%`,
              }}
            />
          </ProgressBar>
        )}
      </div>
      <div className="mr-2">
        <Typography
          as="div"
          className="text-primary text-left break-all"
          weight="medium"
        >
          {title}
        </Typography>
        <Typography
          as="div"
          size="sm"
          className={clsx(
            "text-left",
            {
              complete: "text-success",
              current: "text-accent",
              upcoming: "text-tertiary",
            }[status]
          )}
          weight="medium"
        >
          {
            {
              complete: "Completed",
              current: "In progress",
              upcoming: "Pending",
            }[status]
          }
        </Typography>
      </div>
    </li>
  );
}
