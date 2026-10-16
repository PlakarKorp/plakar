import { ComponentProps } from "react";

import { IllustrationType } from "./illustration.constants";
import DefaultNutNestor from "../../assets/nestor-default-nut.svg";
import DefaultNestor from "../../assets/nestor-default.svg";
import NestorEmperor from "../../assets/nestor-emperor.svg";
import NestorJugglesNuts from "../../assets/nestor-juggles-nuts.svg";
import NestorProud from "../../assets/nestor-proud.svg";
import NestorSad from "../../assets/nestor-sad.svg";
import NestorShrug from "../../assets/nestor-shrug.svg";
import NestorWalking from "../../assets/nestor-walking.svg";
import NestorWaving from "../../assets/nestor-waving.svg";

const illustrationMap: Record<IllustrationType, string> = {
  default: DefaultNestor,
  default_nut: DefaultNutNestor,
  juggles_nuts: NestorJugglesNuts,
  walking: NestorWalking,
  proud: NestorProud,
  waving: NestorWaving,
  sad: NestorSad,
  shrug: NestorShrug,
  emperor: NestorEmperor,
};

type IllustrationProps = {
  type: IllustrationType;
} & Omit<ComponentProps<"img">, "src">;

export function Illustration({ type, ...props }: IllustrationProps) {
  return (
    <img alt={`plakar.io ${type}`} src={illustrationMap[type]} {...props} />
  );
}
