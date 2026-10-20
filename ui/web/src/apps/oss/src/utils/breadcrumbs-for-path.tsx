import { Checksum, Path } from "@plakar-ui/common/api/enrich";
import { Breadcrumbs } from "@plakar-ui/common/components";
import { CopyButton } from "@plakar-ui/ui/components";

function browseSearchFor(path: string) {
  return <T extends object>(prev: T) => ({
    ...prev,
    path,
    page: 0,
    preview: undefined,
    view: undefined,
    filename: undefined,
    mime: undefined,
  });
}

export function breadcrumbsForSource(
  identifier: ReturnType<typeof Checksum>,
  source: number,
  scannedDirectory: ReturnType<typeof Path>
) {
  // A source scanned from "/" would otherwise link to "//".
  const base = scannedDirectory.Value.replace(/\/+$/, "");
  const crumbs = [
    // Link to snapshots list
    <Breadcrumbs.Crumb key="snapshots" to="/snapshots">
      Restore points
    </Breadcrumbs.Crumb>,
    // List to the current snapshot on the base directory
    <Breadcrumbs.Crumb
      key="identifier"
      to="/snapshots/$snapshotId/$source/browse"
      params={{ snapshotId: identifier.Value, source }}
      search={browseSearchFor(`${base}/`)}
      monospace
    >
      <div className="flex items-center gap-1">
        {identifier.toHumanString()}
        <CopyButton
          content={identifier.Value}
          tooltip="Copy full restore point ID"
          className="-my-1"
        />
      </div>
    </Breadcrumbs.Crumb>,
  ];

  return crumbs;
}

export function breadcrumbsForPath(
  identifier: ReturnType<typeof Checksum>,
  source: number,
  scannedDirectory: ReturnType<typeof Path>,
  path: string
) {
  let breadcrumbPath = path;
  const crumbs = [];
  // A source scanned from "/" would otherwise link to "//".
  const base = scannedDirectory.Value.replace(/\/+$/, "");

  if (scannedDirectory.Value && path.startsWith(scannedDirectory.Value)) {
    breadcrumbPath = breadcrumbPath.slice(scannedDirectory.Value.length);

    crumbs.push(
      <Breadcrumbs.Crumb
        key="scannedDirectory"
        to="/snapshots/$snapshotId/$source/browse"
        params={{ snapshotId: identifier.Value, source }}
        search={browseSearchFor(`${base}/`)}
      >
        {scannedDirectory.toHumanString()}
      </Breadcrumbs.Crumb>
    );
  } else {
    crumbs.push(
      <Breadcrumbs.Crumb
        key="scannedDirectory"
        to="/snapshots/$snapshotId/$source/browse"
        params={{ snapshotId: identifier.Value, source }}
        search={browseSearchFor("/")}
      >
        /
      </Breadcrumbs.Crumb>
    );
  }

  const rest = breadcrumbPath.split("/").filter((part) => part !== "");

  rest.forEach((part, idx) => {
    crumbs.push(
      <Breadcrumbs.Crumb
        key={`path-part-${idx}`}
        to="/snapshots/$snapshotId/$source/browse"
        params={{ snapshotId: identifier.Value, source }}
        search={browseSearchFor(`${base}/${rest.slice(0, idx + 1).join("/")}`)}
      >
        {part}
      </Breadcrumbs.Crumb>
    );
  });

  return crumbs;
}
