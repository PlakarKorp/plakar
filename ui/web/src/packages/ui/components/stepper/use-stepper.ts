import { useState } from "react";

/**
 * The state of a stepper. A caller that needs the step outside the stepper,
 * e.g. in a modal footer, holds it with this hook and renders `StepperSteps`.
 */
export function useStepper<T extends readonly string[]>(steps: T) {
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

  return {
    steps,
    currentStep,
    currentIndex,
    progress,
    setProgress,
    goToPreviousStep,
    goToNextStep,
  };
}

export type StepperState<T extends readonly string[]> = ReturnType<
  typeof useStepper<T>
>;
