import clsx from "clsx";
import {
  Slider as RASlider,
  SliderThumb as RASliderThumb,
  SliderTrack as RASliderTrack,
} from "react-aria-components";

export type SliderProps = {
  "aria-label": string;
  value: number;
  onChange: (value: number) => void;
  onChangeEnd?: (value: number) => void;
  minValue?: number;
  maxValue?: number;
  step?: number;
  isDisabled?: boolean;
  className?: string;
};

export function Slider({
  value,
  onChange,
  onChangeEnd,
  minValue = 0,
  maxValue = 100,
  step = 1,
  isDisabled,
  className,
  ...props
}: SliderProps) {
  const isEmptyRange = maxValue <= minValue;
  return (
    <RASlider
      aria-label={props["aria-label"]}
      value={value}
      onChange={onChange}
      onChangeEnd={onChangeEnd}
      minValue={minValue}
      maxValue={isEmptyRange ? minValue + step : maxValue}
      step={step}
      isDisabled={isDisabled || isEmptyRange}
      className={clsx("flex items-center", className)}
    >
      <RASliderTrack className="group relative h-4 w-full cursor-pointer data-disabled:cursor-default">
        {({ state }) => (
          <>
            <div className="bg-tertiary absolute top-1/2 h-1 w-full -translate-y-1/2 rounded-full" />
            <div
              className="bg-accent absolute top-1/2 h-1 -translate-y-1/2 rounded-full group-data-disabled:opacity-50"
              style={{ width: `${state.getThumbPercent(0) * 100}%` }}
            />
            <RASliderThumb
              className={clsx(
                "bg-accent top-1/2 size-3 rounded-full shadow",
                "data-focus-visible:outline-focus data-dragging:size-3.5 data-focus-visible:outline-2 data-focus-visible:outline-offset-2",
                "data-disabled:hidden"
              )}
            />
          </>
        )}
      </RASliderTrack>
    </RASlider>
  );
}
