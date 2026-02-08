const categoryColorMap: Record<string, string> = {
  core: "bg-blue-500/10 text-blue-500",
  bootloader: "bg-orange-500/10 text-orange-500",
  init: "bg-green-500/10 text-green-500",
  systemd: "bg-emerald-500/10 text-emerald-500",
  network: "bg-cyan-500/10 text-cyan-500",
  dns: "bg-teal-500/10 text-teal-500",
  storage: "bg-amber-500/10 text-amber-500",
  device: "bg-yellow-500/10 text-yellow-500",
  user: "bg-indigo-500/10 text-indigo-500",
  extensions: "bg-violet-500/10 text-violet-500",
  tools: "bg-slate-500/10 text-slate-500",
  runtime: "bg-purple-500/10 text-purple-500",
  security: "bg-red-500/10 text-red-500",
  desktop: "bg-pink-500/10 text-pink-500",
  container: "bg-sky-500/10 text-sky-500",
  virtualization: "bg-fuchsia-500/10 text-fuchsia-500",
  toolchain: "bg-lime-500/10 text-lime-500",
  filesystem: "bg-stone-500/10 text-stone-500",
};

export function getCategoryColor(category: string): string {
  return categoryColorMap[category] || "bg-muted text-muted-foreground";
}
