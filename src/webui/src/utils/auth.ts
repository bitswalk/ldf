import type { UserInfo } from "../services/auth";

export function isAdmin(user: UserInfo | null | undefined): boolean {
  return user?.role === "root";
}
