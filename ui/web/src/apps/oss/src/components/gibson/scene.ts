import * as THREE from "three";
import { EffectComposer } from "three/addons/postprocessing/EffectComposer.js";
import { OutputPass } from "three/addons/postprocessing/OutputPass.js";
import { RenderPass } from "three/addons/postprocessing/RenderPass.js";
import { UnrealBloomPass } from "three/addons/postprocessing/UnrealBloomPass.js";

import { Tower, TowerKind, World } from "./world";

export type Telemetry = {
  x: number;
  y: number;
  z: number;
  /** Compass heading in degrees, 0 when flying along -Z. */
  heading: number;
  /** Whether the mouse steers the flight. */
  locked: boolean;
  /** Whether a headset is showing the scene. */
  vr: boolean;
};

export type SceneCallbacks = {
  onTarget: (tower: Tower | null) => void;
  onEnter: (tower: Tower) => void;
  onUp: () => void;
  onExit: () => void;
  onTelemetry: (telemetry: Telemetry) => void;
};

// Pale glass with a green cast, like the Gibson's towers.
const COLORS: Record<TowerKind, number> = {
  snapshot: 0xc8fff0,
  directory: 0x9cfadf,
  file: 0x6fe3c6,
  symlink: 0xd9a6ff,
  other: 0x8aa0a8,
};
const ERROR_COLOR = 0xff3040;
const TARGET_COLOR = 0xffffff;

/** Units per second, and the multiplier while shift is held. */
const SPEED = 24;
const BOOST = 3;
/** Radians per pixel of mouse travel. */
const LOOK = 0.0022;
/** How far a tower can be targeted from. */
const REACH = 500;
const UP = new THREE.Vector3(0, 1, 0);
const CENTER = new THREE.Vector2(0, 0);

type TowerObject = {
  tower: Tower;
  group: THREE.Group;
  glass: THREE.Mesh<THREE.BoxGeometry, THREE.MeshBasicMaterial>;
  stream: THREE.Mesh<THREE.BoxGeometry, THREE.MeshBasicMaterial>;
  edges: THREE.LineSegments<THREE.EdgesGeometry, THREE.LineBasicMaterial>;
  label: THREE.Sprite;
  cap: THREE.Mesh<THREE.PlaneGeometry, THREE.MeshBasicMaterial>;
  color: number;
};

/** The floating data card beside the targeted tower. */
function makeCard(lines: string[], color: number) {
  const canvas = document.createElement("canvas");
  canvas.width = 640;
  canvas.height = 48 + lines.length * 44;
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("2d canvas unavailable");
  }
  ctx.fillStyle = "rgba(0, 8, 16, 0.75)";
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  ctx.strokeStyle = hex(color);
  ctx.lineWidth = 4;
  ctx.strokeRect(2, 2, canvas.width - 4, canvas.height - 4);
  ctx.font = "bold 32px monospace";
  ctx.textBaseline = "top";
  ctx.fillStyle = hex(color);
  lines.forEach((line, i) => {
    ctx.fillText(line.slice(0, 30), 24, 24 + i * 44);
  });

  const texture = new THREE.CanvasTexture(canvas);
  texture.colorSpace = THREE.SRGBColorSpace;
  const sprite = new THREE.Sprite(
    new THREE.SpriteMaterial({
      map: texture,
      transparent: true,
      depthTest: false,
    })
  );
  sprite.scale.set(16, (16 * canvas.height) / canvas.width, 1);
  sprite.renderOrder = 10;
  return sprite;
}

type Flight = {
  fromPos: THREE.Vector3;
  toPos: THREE.Vector3;
  fromQuat: THREE.Quaternion;
  toQuat: THREE.Quaternion;
  start: number;
  ms: number;
  /** Accelerate into the target and fade to black, instead of gliding to it. */
  warp: boolean;
  done: () => void;
};

const FOV = 70;

function easeInOut(t: number) {
  return t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
}

function hex(color: number) {
  return `#${color.toString(16).padStart(6, "0")}`;
}

/** The glyph soup that scrolls along the towers, shared by all of them. */
function makeStreamTexture() {
  const canvas = document.createElement("canvas");
  canvas.width = 256;
  canvas.height = 1024;
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("2d canvas unavailable");
  }
  ctx.font = "bold 20px monospace";
  ctx.textBaseline = "top";
  const glyphs = "0123456789ABCDEF  ";
  for (let y = 4; y < canvas.height; y += 26) {
    let line = "";
    for (let i = 0; i < 20; i++) {
      line += glyphs.charAt(Math.floor(Math.random() * glyphs.length));
    }
    ctx.fillStyle = `rgba(255,255,255,${0.3 + Math.random() * 0.7})`;
    ctx.fillText(line, 8, y);
  }
  const texture = new THREE.CanvasTexture(canvas);
  texture.wrapS = THREE.RepeatWrapping;
  texture.wrapT = THREE.RepeatWrapping;
  texture.colorSpace = THREE.SRGBColorSpace;
  return texture;
}

function makeLabel(text: string, color: number) {
  const canvas = document.createElement("canvas");
  canvas.width = 1024;
  canvas.height = 128;
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("2d canvas unavailable");
  }
  // No glow here: the bloom pass adds it, and a blurred glyph bleeds into a bar.
  ctx.font = "bold 72px monospace";
  ctx.textAlign = "center";
  ctx.textBaseline = "middle";
  ctx.fillStyle = hex(color);
  const shown = text.length > 22 ? `${text.slice(0, 21)}…` : text;
  ctx.fillText(shown, canvas.width / 2, canvas.height / 2);

  const texture = new THREE.CanvasTexture(canvas);
  texture.colorSpace = THREE.SRGBColorSpace;
  const sprite = new THREE.Sprite(
    new THREE.SpriteMaterial({
      map: texture,
      transparent: true,
      depthWrite: false,
    })
  );
  sprite.scale.set(12, 1.5, 1);
  return sprite;
}

/**
 * The ground of the Gibson is a motherboard: traces that bend at right
 * angles between vias, and chips with their pins. One tile, repeated.
 */
function makeMotherboard(anisotropy: number) {
  const size = 1024;
  const canvas = document.createElement("canvas");
  canvas.width = size;
  canvas.height = size;
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("2d canvas unavailable");
  }
  ctx.fillStyle = "#020a10";
  ctx.fillRect(0, 0, size, size);

  // Faint copper pour underneath.
  ctx.strokeStyle = "rgba(6, 44, 56, 0.35)";
  ctx.lineWidth = 1;
  for (let i = 0; i <= size; i += 64) {
    ctx.beginPath();
    ctx.moveTo(i, 0);
    ctx.lineTo(i, size);
    ctx.moveTo(0, i);
    ctx.lineTo(size, i);
    ctx.stroke();
  }

  const step = 32;
  const snap = (v: number) => Math.round(v / step) * step;
  const via = (x: number, y: number) => {
    ctx.beginPath();
    ctx.arc(x, y, 5, 0, Math.PI * 2);
    ctx.fillStyle = "#020a10";
    ctx.fill();
    ctx.strokeStyle = "#0b6a78";
    ctx.lineWidth = 2;
    ctx.stroke();
  };

  // Traces: a walk of axis-aligned segments between two vias.
  ctx.lineCap = "round";
  for (let i = 0; i < 140; i++) {
    let x = snap(Math.random() * size);
    let y = snap(Math.random() * size);
    ctx.strokeStyle = i % 5 === 0 ? "#0a5e6c" : "#053c46";
    ctx.lineWidth = i % 5 === 0 ? 4 : 3;
    ctx.beginPath();
    ctx.moveTo(x, y);
    via(x, y);
    ctx.beginPath();
    ctx.moveTo(x, y);
    const turns = 2 + Math.floor(Math.random() * 4);
    for (let t = 0; t < turns; t++) {
      const run = step * (1 + Math.floor(Math.random() * 6));
      if (t % 2 === 0) {
        x += Math.random() < 0.5 ? -run : run;
      } else {
        y += Math.random() < 0.5 ? -run : run;
      }
      ctx.lineTo(x, y);
    }
    ctx.stroke();
    via(x, y);
  }

  // Chips: dark packages with a notch and rows of pins.
  for (let i = 0; i < 9; i++) {
    const w = step * (3 + Math.floor(Math.random() * 4));
    const h = step * (2 + Math.floor(Math.random() * 3));
    const x = snap(Math.random() * (size - w));
    const y = snap(Math.random() * (size - h));
    ctx.fillStyle = "#04141c";
    ctx.fillRect(x, y, w, h);
    ctx.strokeStyle = "#0d7484";
    ctx.lineWidth = 2;
    ctx.strokeRect(x, y, w, h);
    ctx.fillStyle = "#0d7484";
    for (let px = x + 12; px < x + w - 8; px += 12) {
      ctx.fillRect(px, y - 8, 4, 8);
      ctx.fillRect(px, y + h, 4, 8);
    }
    ctx.beginPath();
    ctx.arc(x + 10, y + 10, 3, 0, Math.PI * 2);
    ctx.fill();
  }

  const texture = new THREE.CanvasTexture(canvas);
  texture.wrapS = THREE.RepeatWrapping;
  texture.wrapT = THREE.RepeatWrapping;
  texture.repeat.set(80, 80);
  texture.anisotropy = anisotropy;
  texture.colorSpace = THREE.SRGBColorSpace;

  const board = new THREE.Mesh(
    new THREE.PlaneGeometry(4000, 4000),
    new THREE.MeshBasicMaterial({ map: texture })
  );
  board.rotation.x = -Math.PI / 2;
  board.position.y = -0.05;
  return board;
}

function makeStars() {
  const count = 1500;
  const positions = new Float32Array(count * 3);
  for (let i = 0; i < count; i++) {
    const theta = Math.random() * Math.PI * 2;
    const phi = Math.acos(Math.random());
    const r = 1200;
    positions[i * 3] = r * Math.sin(phi) * Math.cos(theta);
    positions[i * 3 + 1] = r * Math.cos(phi);
    positions[i * 3 + 2] = r * Math.sin(phi) * Math.sin(theta);
  }
  const geometry = new THREE.BufferGeometry();
  geometry.setAttribute("position", new THREE.BufferAttribute(positions, 3));
  const material = new THREE.PointsMaterial({
    color: 0x9fb8ff,
    size: 2,
    sizeAttenuation: false,
    fog: false,
  });
  return new THREE.Points(geometry, material);
}

/**
 * The Gibson: a city of glass towers on a grid, flown through with the
 * keyboard and the mouse. It renders into the container it is given and
 * reports what the pilot does through the callbacks.
 */
export class GibsonScene {
  private readonly container: HTMLElement;
  private readonly callbacks: SceneCallbacks;
  private readonly renderer: THREE.WebGLRenderer;
  private readonly scene = new THREE.Scene();
  private readonly camera: THREE.PerspectiveCamera;
  /** Moves the pilot; the camera hangs from it, so a headset can steer the camera later. */
  private readonly rig = new THREE.Group();
  private readonly composer: EffectComposer;
  private readonly bloom: UnrealBloomPass;
  private readonly city = new THREE.Group();
  private readonly streamTexture: THREE.CanvasTexture;
  private readonly raycaster = new THREE.Raycaster();
  private readonly clock = new THREE.Clock();
  private readonly keys = new Set<string>();
  private readonly overlay: HTMLDivElement;
  private readonly resizeObserver: ResizeObserver;
  private readonly forward = new THREE.Vector3();
  private readonly right = new THREE.Vector3();
  private readonly velocity = new THREE.Vector3();
  private readonly origin = new THREE.Vector3();
  private readonly direction = new THREE.Vector3();
  private readonly orientation = new THREE.Quaternion();

  /** The panel read inside the headset, where the DOM HUD cannot follow. */
  private readonly hudCanvas = document.createElement("canvas");
  private readonly hudTexture = new THREE.CanvasTexture(this.hudCanvas);
  private readonly hudPanel = new THREE.Mesh(
    new THREE.PlaneGeometry(0.6, 0.316),
    new THREE.MeshBasicMaterial({ map: this.hudTexture, transparent: true })
  );
  /** The ray out of the aiming hand. */
  private readonly beam = new THREE.Line(
    new THREE.BufferGeometry().setFromPoints([
      new THREE.Vector3(0, 0, 0),
      new THREE.Vector3(0, 0, -1),
    ]),
    new THREE.LineBasicMaterial({
      color: 0x8ff6ff,
      transparent: true,
      opacity: 0.5,
    })
  );
  private readonly pressed = new Set<string>();
  private turned = false;
  private frozen = false;
  /** Where the mouse points, in clip space, while it is not locked. */
  private readonly pointer = new THREE.Vector2(0, 0);
  private pointerInside = false;

  private towers: TowerObject[] = [];
  private glasses: THREE.Object3D[] = [];
  private targeted: TowerObject | null = null;
  private yaw = 0;
  private pitch = 0;
  private locked = false;
  private dragging = false;
  private dragTravel = 0;
  private flight: Flight | null = null;
  private telemetryClock = 0;
  private card: THREE.Sprite | null = null;
  private alarmClock = 0;

  constructor(container: HTMLElement, callbacks: SceneCallbacks) {
    this.container = container;
    this.callbacks = callbacks;

    const width = container.clientWidth || 1;
    const height = container.clientHeight || 1;

    this.renderer = new THREE.WebGLRenderer({
      antialias: true,
      powerPreference: "high-performance",
    });
    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 1.5));
    this.renderer.setSize(width, height);
    this.renderer.domElement.style.display = "block";
    container.appendChild(this.renderer.domElement);

    this.scene.background = new THREE.Color(0x000008);
    this.scene.fog = new THREE.FogExp2(0x000010, 0.0032);

    this.camera = new THREE.PerspectiveCamera(FOV, width / height, 0.1, 3000);
    this.rig.add(this.camera);
    this.scene.add(this.rig);
    this.scene.add(this.city);

    this.scene.add(
      makeMotherboard(this.renderer.capabilities.getMaxAnisotropy())
    );
    this.scene.add(makeStars());

    this.streamTexture = makeStreamTexture();

    this.composer = new EffectComposer(this.renderer);
    this.composer.addPass(new RenderPass(this.scene, this.camera));
    this.bloom = new UnrealBloomPass(
      new THREE.Vector2(width, height),
      0.7,
      0.4,
      0.2
    );
    this.composer.addPass(this.bloom);
    // Back to sRGB: without it the composer's output stays linear and washed out.
    this.composer.addPass(new OutputPass());

    this.setupXR();

    // Black between two levels, so the swap reads as a jump.
    this.overlay = document.createElement("div");
    Object.assign(this.overlay.style, {
      position: "absolute",
      inset: "0",
      background: "#000",
      opacity: "0",
      pointerEvents: "none",
      transition: "opacity 0.6s ease-out",
    });
    container.appendChild(this.overlay);

    window.addEventListener("keydown", this.onKeyDown);
    window.addEventListener("keyup", this.onKeyUp);
    window.addEventListener("blur", this.onBlur);
    document.addEventListener("mousemove", this.onMouseMove);
    document.addEventListener("mouseup", this.onMouseUp);
    document.addEventListener("pointerlockchange", this.onLockChange);
    this.renderer.domElement.addEventListener("mousedown", this.onMouseDown);

    this.resizeObserver = new ResizeObserver(() => this.resize());
    this.resizeObserver.observe(container);

    this.renderer.setAnimationLoop(() => this.frame());
  }

  dispose() {
    this.renderer.setAnimationLoop(null);
    this.resizeObserver.disconnect();
    window.removeEventListener("keydown", this.onKeyDown);
    window.removeEventListener("keyup", this.onKeyUp);
    window.removeEventListener("blur", this.onBlur);
    document.removeEventListener("mousemove", this.onMouseMove);
    document.removeEventListener("mouseup", this.onMouseUp);
    document.removeEventListener("pointerlockchange", this.onLockChange);
    this.renderer.domElement.removeEventListener("mousedown", this.onMouseDown);
    if (document.pointerLockElement === this.renderer.domElement) {
      document.exitPointerLock();
    }
    this.renderer.xr.removeEventListener("sessionstart", this.onSessionStart);
    this.renderer.xr.removeEventListener("sessionend", this.onSessionEnd);
    void this.renderer.xr.getSession()?.end();
    this.clearTowers();
    this.hudTexture.dispose();
    this.streamTexture.dispose();
    this.composer.dispose();
    this.renderer.dispose();
    this.renderer.domElement.remove();
    this.overlay.remove();
  }

  /** Hands the mouse to the flight controls. */
  requestLock() {
    this.renderer.domElement.requestPointerLock();
  }

  /** Stops the controls while something else, such as a file, has the keyboard. */
  setFrozen(frozen: boolean) {
    this.frozen = frozen;
    this.keys.clear();
    this.dragging = false;
  }

  /** Whether this browser can open an immersive VR session. */
  static async vrSupported() {
    const xr = navigator.xr;
    if (!xr) {
      return false;
    }
    try {
      return await xr.isSessionSupported("immersive-vr");
    } catch {
      return false;
    }
  }

  /** Opens the VR session: the headset then steers the camera and the controllers fly. */
  async enterVR() {
    const xr = navigator.xr;
    if (!xr) {
      throw new Error("WebXR is not available in this browser");
    }
    const session = await xr.requestSession("immersive-vr", {
      optionalFeatures: ["local-floor", "bounded-floor"],
    });
    await this.renderer.xr.setSession(session);
  }

  /** Redraws the in-headset panel, one entry per line. */
  setHud(lines: string[]) {
    const ctx = this.hudCanvas.getContext("2d");
    if (!ctx) {
      return;
    }
    const { width, height } = this.hudCanvas;
    ctx.clearRect(0, 0, width, height);
    ctx.fillStyle = "rgba(0, 6, 20, 0.85)";
    ctx.fillRect(0, 0, width, height);
    ctx.strokeStyle = "rgba(0, 240, 255, 0.7)";
    ctx.lineWidth = 4;
    ctx.strokeRect(2, 2, width - 4, height - 4);
    ctx.font = "bold 30px monospace";
    ctx.fillStyle = "#8ff6ff";
    ctx.textBaseline = "top";
    lines.slice(0, 14).forEach((line, i) => {
      ctx.fillText(line.slice(0, 52), 24, 20 + i * 36);
    });
    this.hudTexture.needsUpdate = true;
  }

  private setupXR() {
    this.renderer.xr.enabled = true;
    for (const index of [0, 1]) {
      const grip = this.renderer.xr.getControllerGrip(index);
      grip.add(
        new THREE.Mesh(
          new THREE.BoxGeometry(0.04, 0.04, 0.1),
          new THREE.MeshBasicMaterial({ color: 0x8ff6ff, wireframe: true })
        )
      );
      this.rig.add(grip);
      this.rig.add(this.renderer.xr.getController(index));
    }
    this.beam.scale.z = REACH;
    this.beam.visible = false;
    this.rig.add(this.beam);

    this.hudCanvas.width = 1024;
    this.hudCanvas.height = 540;
    this.hudTexture.colorSpace = THREE.SRGBColorSpace;
    this.hudPanel.position.set(0, -0.3, -0.8);
    this.hudPanel.rotation.x = 0.35;
    this.hudPanel.visible = false;
    this.camera.add(this.hudPanel);
    this.setHud(["THE GIBSON"]);

    this.renderer.xr.addEventListener("sessionstart", this.onSessionStart);
    this.renderer.xr.addEventListener("sessionend", this.onSessionEnd);
  }

  private readonly onSessionStart = () => {
    this.hudPanel.visible = true;
    this.beam.visible = true;
    this.emitTelemetry();
  };

  private readonly onSessionEnd = () => {
    this.hudPanel.visible = false;
    this.beam.visible = false;
    this.pressed.clear();
    this.emitTelemetry();
  };

  /** Replaces the city and puts the pilot at its edge, looking at it. */
  setWorld(world: World) {
    this.clearTowers();
    for (const tower of world.towers) {
      const object = this.buildTower(tower);
      this.towers.push(object);
      this.glasses.push(object.glass);
      this.city.add(object.group);
    }

    this.flight = null;
    this.rig.position.set(0, 10 + world.extent * 0.2, world.extent + 28);
    this.yaw = 0;
    this.pitch = -0.2;
    this.applyLook();
    this.camera.fov = FOV;
    this.camera.updateProjectionMatrix();

    this.overlay.style.transition = "none";
    this.overlay.style.opacity = "1";
    // Reflow, so the transition below starts from black.
    void this.overlay.offsetHeight;
    this.overlay.style.transition = "opacity 0.6s ease-out";
    this.overlay.style.opacity = "0";

    this.emitTelemetry();
  }

  private clearTowers() {
    for (const object of this.towers) {
      this.city.remove(object.group);
      object.glass.geometry.dispose();
      object.glass.material.dispose();
      object.stream.material.dispose();
      object.edges.geometry.dispose();
      object.edges.material.dispose();
      object.label.material.map?.dispose();
      object.label.material.dispose();
      object.cap.geometry.dispose();
      object.cap.material.dispose();
    }
    this.towers = [];
    this.glasses = [];
    this.dropCard();
    if (this.targeted) {
      this.targeted = null;
      this.callbacks.onTarget(null);
    }
  }

  /**
   * Paints new colors and details onto the towers already standing, for a
   * level whose verification just came back. The camera does not move.
   */
  restyle(world: World) {
    const towers = new Map(world.towers.map((tower) => [tower.id, tower]));
    for (const object of this.towers) {
      const tower = towers.get(object.tower.id);
      if (!tower) {
        continue;
      }
      object.tower = tower;
      const color = tower.hasErrors ? ERROR_COLOR : COLORS[tower.kind];
      if (color === object.color) {
        continue;
      }
      object.color = color;
      object.glass.material.color.set(color);
      object.stream.material.color.set(color);
      object.cap.material.color.set(color);
      object.edges.material.color.set(
        object === this.targeted ? TARGET_COLOR : color
      );
      if (!tower.hasErrors) {
        object.edges.material.opacity = 0.9;
        object.cap.material.opacity = 0.55;
        object.stream.material.opacity = 0.8;
      }
      const label = makeLabel(tower.label, color);
      label.position.copy(object.label.position);
      label.scale.copy(object.label.scale);
      object.group.remove(object.label);
      object.label.material.map?.dispose();
      object.label.material.dispose();
      object.group.add(label);
      object.label = label;
    }
    if (this.targeted) {
      this.highlight(this.targeted, true);
      this.callbacks.onTarget(this.targeted.tower);
    }
  }

  private dropCard() {
    if (!this.card) {
      return;
    }
    this.city.remove(this.card);
    this.card.material.map?.dispose();
    this.card.material.dispose();
    this.card = null;
  }

  private buildTower(tower: Tower): TowerObject {
    const color = tower.hasErrors ? ERROR_COLOR : COLORS[tower.kind];
    const geometry = new THREE.BoxGeometry(
      tower.width,
      tower.height,
      tower.depth
    );

    // The glyph stream tiles every 4 units instead of stretching over the tower.
    const uv = geometry.getAttribute("uv");
    const horizontal = Math.max(tower.width, tower.depth) / 4;
    const vertical = tower.height / 4;
    for (let i = 0; i < uv.count; i++) {
      uv.setXY(i, uv.getX(i) * horizontal, uv.getY(i) * vertical);
    }
    uv.needsUpdate = true;

    const glass = new THREE.Mesh(
      geometry,
      new THREE.MeshBasicMaterial({
        color,
        transparent: true,
        opacity: 0.16,
        depthWrite: false,
      })
    );
    const stream = new THREE.Mesh(
      geometry,
      new THREE.MeshBasicMaterial({
        map: this.streamTexture,
        color,
        transparent: true,
        opacity: 0.8,
        depthWrite: false,
        blending: THREE.AdditiveBlending,
      })
    );
    const edges = new THREE.LineSegments(
      new THREE.EdgesGeometry(geometry),
      new THREE.LineBasicMaterial({ color, transparent: true, opacity: 0.9 })
    );
    const label = makeLabel(tower.label, color);
    label.position.y = tower.height / 2 + 1.6;

    // A lit roof, the brightest part of the tower, as in the film.
    const cap = new THREE.Mesh(
      new THREE.PlaneGeometry(tower.width, tower.depth),
      new THREE.MeshBasicMaterial({
        color,
        transparent: true,
        opacity: 0.55,
        side: THREE.DoubleSide,
        depthWrite: false,
      })
    );
    cap.rotation.x = -Math.PI / 2;
    cap.position.y = tower.height / 2;

    const group = new THREE.Group();
    group.add(glass, stream, edges, label, cap);
    group.position.set(tower.x, tower.height / 2, tower.z);

    const object: TowerObject = {
      tower,
      group,
      glass,
      stream,
      edges,
      label,
      cap,
      color,
    };
    glass.userData.tower = object;
    return object;
  }

  private readonly onKeyDown = (event: KeyboardEvent) => {
    if (this.frozen) {
      return;
    }
    switch (event.code) {
      case "Enter":
        if (!event.repeat) {
          this.activate();
        }
        break;
      case "Backspace":
        if (!event.repeat) {
          this.ascend();
        }
        break;
      case "KeyX":
        this.callbacks.onExit();
        break;
      case "KeyL":
        this.requestLock();
        break;
      default:
        this.keys.add(event.code);
    }
    if (
      event.code === "Space" ||
      event.code === "Backspace" ||
      event.code.startsWith("Arrow")
    ) {
      event.preventDefault();
    }
  };

  private readonly onKeyUp = (event: KeyboardEvent) => {
    this.keys.delete(event.code);
  };

  private readonly onBlur = () => {
    this.keys.clear();
  };

  // Without pointer lock the mouse steers while a button is held, so a
  // trackpad or an embedded browser still flies.
  private readonly onMouseDown = () => {
    if (this.frozen) {
      return;
    }
    if (this.locked) {
      this.activate();
      return;
    }
    this.dragging = true;
    this.dragTravel = 0;
  };

  private readonly onMouseUp = () => {
    if (!this.dragging) {
      return;
    }
    this.dragging = false;
    // A click, not a drag: dive into the tower under the mouse.
    if (this.dragTravel < 6 && this.targeted) {
      this.activate();
    }
  };

  private readonly onMouseMove = (event: MouseEvent) => {
    if (!this.locked) {
      const rect = this.renderer.domElement.getBoundingClientRect();
      this.pointer.set(
        ((event.clientX - rect.left) / rect.width) * 2 - 1,
        -((event.clientY - rect.top) / rect.height) * 2 + 1
      );
      this.pointerInside =
        event.clientX >= rect.left &&
        event.clientX <= rect.right &&
        event.clientY >= rect.top &&
        event.clientY <= rect.bottom;
    }
    if (this.flight || !(this.locked || this.dragging)) {
      return;
    }
    this.dragTravel += Math.abs(event.movementX) + Math.abs(event.movementY);
    this.yaw -= event.movementX * LOOK;
    this.pitch = THREE.MathUtils.clamp(
      this.pitch - event.movementY * LOOK,
      -1.5,
      1.5
    );
    this.applyLook();
  };

  private readonly onLockChange = () => {
    this.locked = document.pointerLockElement === this.renderer.domElement;
    this.emitTelemetry();
  };

  private applyLook() {
    this.camera.quaternion.setFromEuler(
      new THREE.Euler(this.pitch, this.yaw, 0, "YXZ")
    );
  }

  /** Dives into the targeted tower, then lets the pilot know. */
  private activate() {
    if (this.flight || !this.targeted) {
      return;
    }
    const object = this.targeted;
    const tower = object.tower;
    const center = new THREE.Vector3(tower.x, tower.height / 2, tower.z);
    if (tower.destination) {
      // Sucked straight into the tower's heart.
      this.flyTo(center, 650, () => this.callbacks.onEnter(tower), true);
      return;
    }
    if (!tower.file) {
      this.pulse(object);
      return;
    }
    // A file: pulled toward it, stopping short, then it opens.
    const toPos = this.rig.position
      .clone()
      .sub(center)
      .setLength(tower.width + 6)
      .add(center);
    this.flyTo(toPos, 450, () => this.callbacks.onEnter(tower));
  }

  /** Sucked straight up out of the level, then lets the pilot know. */
  private ascend() {
    if (this.flight) {
      return;
    }
    const toPos = this.rig.position.clone().add(new THREE.Vector3(0, 80, 0));
    this.flyTo(toPos, 500, () => this.callbacks.onUp(), true);
  }

  /**
   * Moves the pilot to a point without turning the head: every flight is a
   * pull along a straight line. A warp accelerates into it and blacks out.
   */
  private flyTo(
    toPos: THREE.Vector3,
    ms: number,
    done: () => void,
    warp = false
  ) {
    this.flight = {
      fromPos: this.rig.position.clone(),
      toPos,
      fromQuat: this.camera.quaternion.clone(),
      toQuat: this.camera.quaternion.clone(),
      start: performance.now(),
      ms,
      warp,
      done,
    };
  }

  /** A flash on a tower that cannot be entered. */
  private pulse(object: TowerObject) {
    object.glass.material.opacity = 0.6;
    setTimeout(() => {
      object.glass.material.opacity = object === this.targeted ? 0.35 : 0.16;
    }, 120);
  }

  private resize() {
    const width = this.container.clientWidth || 1;
    const height = this.container.clientHeight || 1;
    this.camera.aspect = width / height;
    this.camera.updateProjectionMatrix();
    this.renderer.setSize(width, height);
    this.composer.setSize(width, height);
    this.bloom.resolution.set(width, height);
  }

  private frame() {
    const dt = Math.min(this.clock.getDelta(), 0.1);
    this.streamTexture.offset.y -= dt * 0.08;
    this.pulseAlarms(dt);

    let aimed = false;
    if (this.flight) {
      this.advanceFlight();
    } else if (this.renderer.xr.isPresenting) {
      aimed = this.pollControllers(dt);
    } else if (!this.frozen) {
      this.move(dt);
    }
    if (!aimed) {
      const from = this.locked || !this.pointerInside ? CENTER : this.pointer;
      this.raycaster.setFromCamera(from, this.camera);
      this.pickTarget();
    }

    this.telemetryClock += dt;
    if (this.telemetryClock > 0.1) {
      this.telemetryClock = 0;
      this.emitTelemetry();
    }

    if (this.renderer.xr.isPresenting) {
      this.renderer.render(this.scene, this.camera);
    } else {
      this.composer.render();
    }
  }

  private advanceFlight() {
    const flight = this.flight;
    if (!flight) {
      return;
    }
    const t = Math.min((performance.now() - flight.start) / flight.ms, 1);
    const k = flight.warp ? t * t * t : easeInOut(t);
    this.rig.position.lerpVectors(flight.fromPos, flight.toPos, k);
    this.camera.quaternion.slerpQuaternions(flight.fromQuat, flight.toQuat, k);
    if (flight.warp) {
      // The view stretches as the speed builds, then blacks out at the plunge.
      this.camera.fov = FOV + 60 * k;
      this.camera.updateProjectionMatrix();
      this.overlay.style.transition = "none";
      this.overlay.style.opacity = String(Math.max(0, (t - 0.6) / 0.4));
    }
    if (t >= 1) {
      this.flight = null;
      const euler = new THREE.Euler().setFromQuaternion(
        this.camera.quaternion,
        "YXZ"
      );
      this.yaw = euler.y;
      this.pitch = euler.x;
      flight.done();
    }
  }

  private move(dt: number) {
    const keys = this.keys;
    const boost = keys.has("ShiftLeft") || keys.has("ShiftRight");
    const step = SPEED * (boost ? BOOST : 1) * dt;

    this.camera.getWorldDirection(this.forward);
    this.right.crossVectors(this.forward, UP).normalize();
    const v = this.velocity.set(0, 0, 0);

    if (keys.has("KeyW") || keys.has("ArrowUp")) {
      v.add(this.forward);
    }
    if (keys.has("KeyS") || keys.has("ArrowDown")) {
      v.sub(this.forward);
    }
    if (keys.has("KeyD") || keys.has("ArrowRight")) {
      v.add(this.right);
    }
    if (keys.has("KeyA") || keys.has("ArrowLeft")) {
      v.sub(this.right);
    }
    if (keys.has("Space") || keys.has("KeyR")) {
      v.y += 1;
    }
    if (keys.has("KeyC") || keys.has("KeyF")) {
      v.y -= 1;
    }
    if (v.lengthSq() > 0) {
      this.rig.position.addScaledVector(v.normalize(), step);
    }
    if (this.rig.position.y < 0.5) {
      this.rig.position.y = 0.5;
    }
  }

  /**
   * Reads the controllers: the left stick flies where the head looks, the
   * right stick climbs and snap-turns, the right trigger dives, B surfaces,
   * Y leaves VR. Returns whether a hand aimed, so the crosshair stays off.
   */
  private pollControllers(dt: number) {
    const session = this.renderer.xr.getSession();
    if (!session) {
      return false;
    }
    let aimed = false;
    for (let i = 0; i < session.inputSources.length; i++) {
      const source = session.inputSources[i];
      const pad = source?.gamepad;
      if (!source || !pad) {
        continue;
      }
      const axisX = pad.axes[2] ?? 0;
      const axisY = pad.axes[3] ?? 0;

      if (source.handedness === "left") {
        const boost = pad.buttons[1]?.pressed ? BOOST : 1;
        this.camera.getWorldDirection(this.forward);
        this.right.crossVectors(this.forward, UP).normalize();
        const v = this.velocity.set(0, 0, 0);
        if (Math.abs(axisY) > 0.15) {
          v.addScaledVector(this.forward, -axisY);
        }
        if (Math.abs(axisX) > 0.15) {
          v.addScaledVector(this.right, axisX);
        }
        if (v.lengthSq() > 0) {
          this.rig.position.addScaledVector(v, SPEED * boost * dt);
        }
        if (this.edge("left", 5, pad)) {
          void session.end();
        }
      } else if (source.handedness === "right") {
        const controller = this.renderer.xr.getController(i);
        aimed = true;
        controller.getWorldPosition(this.origin);
        controller.getWorldQuaternion(this.orientation);
        this.direction.set(0, 0, -1).applyQuaternion(this.orientation);
        this.raycaster.set(this.origin, this.direction);
        this.pickTarget();
        this.beam.position.copy(controller.position);
        this.beam.quaternion.copy(controller.quaternion);

        if (Math.abs(axisY) > 0.15) {
          this.rig.position.y -= axisY * SPEED * dt;
        }
        if (Math.abs(axisX) > 0.7) {
          if (!this.turned) {
            this.rig.rotateY(axisX > 0 ? -Math.PI / 6 : Math.PI / 6);
            this.turned = true;
          }
        } else if (Math.abs(axisX) < 0.3) {
          this.turned = false;
        }
        if (this.edge("right", 0, pad)) {
          this.activate();
        }
        if (this.edge("right", 5, pad)) {
          this.ascend();
        }
      }
    }
    if (this.rig.position.y < 0.5) {
      this.rig.position.y = 0.5;
    }
    return aimed;
  }

  /** True on the frame a button goes down. */
  private edge(hand: string, index: number, pad: Gamepad) {
    const key = `${hand}:${index}`;
    const down = pad.buttons[index]?.pressed ?? false;
    const was = this.pressed.has(key);
    if (down && !was) {
      this.pressed.add(key);
      return true;
    }
    if (!down && was) {
      this.pressed.delete(key);
    }
    return false;
  }

  /** Targets the tower along the raycaster. */
  private pickTarget() {
    this.raycaster.far = REACH;
    const hit = this.raycaster.intersectObjects(this.glasses, false)[0];
    const object: TowerObject | null = hit
      ? (hit.object.userData.tower as TowerObject)
      : null;
    if (object === this.targeted) {
      return;
    }
    if (this.targeted) {
      this.highlight(this.targeted, false);
    }
    this.targeted = object;
    if (object) {
      this.highlight(object, true);
    }
    this.renderer.domElement.style.cursor = object ? "pointer" : "crosshair";
    this.callbacks.onTarget(object?.tower ?? null);
  }

  private highlight(object: TowerObject, on: boolean) {
    object.edges.material.color.set(on ? TARGET_COLOR : object.color);
    object.glass.material.opacity = on ? 0.35 : 0.16;
    object.label.scale.set(on ? 15 : 12, on ? 1.9 : 1.5, 1);

    this.dropCard();
    if (on) {
      const tower = object.tower;
      const card = makeCard(
        [tower.label, ...tower.details.slice(1, 4).map(([, value]) => value)],
        object.color
      );
      card.position.set(
        tower.x + tower.width / 2 + 9,
        tower.height + 3,
        tower.z
      );
      this.city.add(card);
      this.card = card;
    }
  }

  /** Red towers breathe, so a corrupted branch catches the eye from afar. */
  private pulseAlarms(dt: number) {
    this.alarmClock += dt;
    const glow = 0.6 + 0.4 * Math.sin(this.alarmClock * 4);
    for (const object of this.towers) {
      if (!object.tower.hasErrors) {
        continue;
      }
      object.edges.material.opacity = 0.5 + 0.5 * glow;
      object.cap.material.opacity = 0.3 + 0.5 * glow;
      object.stream.material.opacity = 0.5 + 0.4 * glow;
    }
  }

  private emitTelemetry() {
    const degrees = (-this.yaw * 180) / Math.PI;
    this.callbacks.onTelemetry({
      x: this.rig.position.x,
      y: this.rig.position.y,
      z: this.rig.position.z,
      heading: ((degrees % 360) + 360) % 360,
      locked: this.locked,
      vr: this.renderer.xr.isPresenting,
    });
  }
}
