import {
  Avatar,
  NavListItemProps,
  Skeleton,
  Typography,
} from "@plakar-ui/ui/components";

export type User = {
  avatarUrl?: string;
  isAuthenticated?: boolean;
  isPending?: boolean;
  name?: string;
  email?: string;
  href?: NavListItemProps["href"];
};

export function UserAvatarImage({ user }: { user: User }) {
  return user.isAuthenticated ? (
    <Avatar
      text={user.name || ""}
      url={user.avatarUrl}
      isLoading={user.isPending}
      status="online"
    />
  ) : (
    <Avatar icon="incognito" isLoading={user.isPending} status="offline" />
  );
}

type UserAvatarProps = {
  user: User;
  isExpanded: boolean;
};

export function UserAvatar({ user, isExpanded }: UserAvatarProps) {
  return (
    <>
      <div className="flex w-[39px] justify-center">
        <UserAvatarImage user={user} />
      </div>
      {isExpanded && (
        <span className="flex min-w-0 flex-col">
          {user.isPending ? (
            <Skeleton variant="text" size="sm" width={135} />
          ) : (
            <Typography
              className="text-tertiary group-data-hovered/user:text-secondary truncate"
              size="sm"
              weight="medium"
            >
              {user.isAuthenticated
                ? user.name || "You are logged in"
                : "Guest"}
            </Typography>
          )}
        </span>
      )}
    </>
  );
}
