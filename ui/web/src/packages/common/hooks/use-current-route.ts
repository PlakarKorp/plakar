import { useMatchRoute } from "@tanstack/react-router";

export function useCurrentRoute(routes: string[]) {
  const matchRoute = useMatchRoute();

  const pendingRoute = routes.find((to) =>
    matchRoute({ to, fuzzy: true, pending: true })
  );
  const currentRoute =
    pendingRoute ?? routes.find((to) => matchRoute({ to, fuzzy: true }));

  return currentRoute;
}
