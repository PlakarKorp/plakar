import { FloatingOverlayButton } from "@plakar-ui/common/components";
import { useLocalStorage } from "@plakar-ui/common/hooks";
import { Icon } from "@plakar-ui/ui/components";
import { Illustration } from "@plakar-ui/ui/components";

type FloatingDemoButtonProps = {
  bottom: string;
};

export function FloatingDemoButton({ bottom }: FloatingDemoButtonProps) {
  const [hasSeenDemoInfo, setHasSeenDemoInfo] = useLocalStorage(
    "plakar_demo_info_seen",
    false
  );

  const handleChange = () => {
    if (!hasSeenDemoInfo) {
      setHasSeenDemoInfo(true);
    }
  };

  return (
    <FloatingOverlayButton
      icon="info-circle"
      defaultOpen={!hasSeenDemoInfo}
      onOpenChange={handleChange}
      bottom={bottom}
      content={
        <div className="relative flex flex-col gap-2 p-4 pb-0">
          <p>
            Welcome to <b>Plakar UI demo</b>!
          </p>
          <p>
            This is a playground with <b>real data</b>, already signed in to a{" "}
            <b>demo account</b>.
          </p>
          <p>
            Want to keep your data safe? Check out{" "}
            <a
              href="https://plakar.io/"
              target="_blank"
              className="text-accent hover:text-accent-hover hover:underline"
            >
              plakar.io
            </a>
            .
          </p>
          <ul className="bg-accent-tertiary -mx-4 mt-1 px-4 py-1">
            <li className="flex items-center gap-2">
              <Icon type="discord" size="md" />
              <p>
                Join us on{" "}
                <a
                  href="https://discord.gg/uuegtnF2Q5"
                  target="_blank"
                  className="text-accent hover:text-accent-hover hover:underline"
                >
                  Discord
                </a>
                !
              </p>
            </li>
            <li className="flex items-center gap-2">
              <Icon type="github" size="md" />{" "}
              <p>
                Star us on{" "}
                <a
                  href="https://github.com/PlakarKorp/plakar"
                  target="_blank"
                  className="text-accent hover:text-accent-hover hover:underline"
                >
                  GitHub
                </a>
                !
              </p>
            </li>
          </ul>
          <Illustration
            type="waving"
            className="absolute -right-8 -bottom-6"
            width={120}
          />
        </div>
      }
    />
  );
}
