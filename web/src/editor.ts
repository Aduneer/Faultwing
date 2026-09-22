export type EditorPreset = "vscode" | "vscode-insiders" | "cursor" | "zed" | "jetbrains" | "custom";

export type EditorSettings = {
  preset: EditorPreset;
  customTemplate: string;
  jetbrainsProduct: string;
};

export type SourceLocation = {
  path: string;
  line: number;
  column: number;
};

export const editorPresets: Array<{ value: EditorPreset; label: string }> = [
  { value: "vscode", label: "Visual Studio Code" },
  { value: "vscode-insiders", label: "Visual Studio Code Insiders" },
  { value: "cursor", label: "Cursor" },
  { value: "zed", label: "Zed" },
  { value: "jetbrains", label: "JetBrains IDE (Toolbox)" },
  { value: "custom", label: "Another editor (custom URL)" },
];

export const jetbrainsProducts = [
  { value: "idea", label: "IntelliJ IDEA" },
  { value: "goland", label: "GoLand" },
  { value: "pycharm", label: "PyCharm" },
  { value: "web-storm", label: "WebStorm" },
  { value: "php-storm", label: "PhpStorm" },
  { value: "rider", label: "Rider" },
  { value: "clion", label: "CLion" },
  { value: "rubymine", label: "RubyMine" },
  { value: "rust-rover", label: "RustRover" },
  { value: "datagrip", label: "DataGrip" },
] as const;

const presetTemplates: Record<"vscode" | "vscode-insiders" | "cursor" | "zed", string> = {
  vscode: "vscode://file/{path}:{line}:{column}",
  "vscode-insiders": "vscode-insiders://file/{path}:{line}:{column}",
  cursor: "cursor://file/{path}:{line}:{column}",
  zed: "zed://file/{pathNoLeadingSlash}:{line}:{column}",
};

const settingsKey = "flytrap.editor.settings";

export function readEditorSettings(): EditorSettings | null {
  try {
    const stored = localStorage.getItem(settingsKey);
    if (!stored) return null;
    const parsed = JSON.parse(stored) as Partial<EditorSettings>;
    if (!editorPresets.some((preset) => preset.value === parsed.preset)) return null;
    const jetbrainsProduct = typeof parsed.jetbrainsProduct === "string"
      && jetbrainsProducts.some((product) => product.value === parsed.jetbrainsProduct)
      ? parsed.jetbrainsProduct
      : "idea";
    return {
      preset: parsed.preset as EditorPreset,
      customTemplate: typeof parsed.customTemplate === "string" ? parsed.customTemplate : "",
      jetbrainsProduct,
    };
  } catch {
    return null;
  }
}

export function saveEditorSettings(settings: EditorSettings): boolean {
  try {
    localStorage.setItem(settingsKey, JSON.stringify(settings));
    return true;
  } catch {
    return false;
  }
}

export function readProjectSourceRoot(projectId: number): string {
  try {
    return localStorage.getItem(`flytrap.editor.root.${projectId}`) ?? "";
  } catch {
    return "";
  }
}

export function saveProjectSourceRoot(projectId: number, root: string): boolean {
  try {
    const key = `flytrap.editor.root.${projectId}`;
    if (root) localStorage.setItem(key, root);
    else localStorage.removeItem(key);
    return true;
  } catch {
    return false;
  }
}

export function readJetBrainsProjectName(projectId: number): string {
  try {
    return localStorage.getItem(`flytrap.editor.jetbrains-project.${projectId}`) ?? "";
  } catch {
    return "";
  }
}

export function saveJetBrainsProjectName(projectId: number, projectName: string): boolean {
  try {
    const key = `flytrap.editor.jetbrains-project.${projectId}`;
    if (projectName) localStorage.setItem(key, projectName);
    else localStorage.removeItem(key);
    return true;
  } catch {
    return false;
  }
}

export function sourceLocationFromStacktrace(stacktrace: string): SourceLocation | null {
  for (const line of stacktrace.split("\n")) {
    const python = line.match(/\bFile\s+["']([^"']+)["'],\s+line\s+(\d+)/);
    if (python) return location(python[1], python[2]);

    const parenthesized = line.match(/\((.+):(\d+)(?::(\d+))?\)\s*$/);
    if (parenthesized) return location(parenthesized[1], parenthesized[2], parenthesized[3]);

    const bare = line.match(/(?:^|\s)((?:[A-Za-z]:[\\/]|\/|\.?\.?[\\/])?[^\s():]+\.[A-Za-z0-9]+):(\d+)(?::(\d+))?\s*$/);
    if (bare) return location(bare[1], bare[2], bare[3]);
  }
  return null;
}

function location(path: string, line: string, column = "1"): SourceLocation {
  return {
    path: path.trim(),
    line: Math.max(1, Number(line)),
    column: Math.max(1, Number(column)),
  };
}

function isAbsolutePath(path: string) {
  return path.startsWith("/") || path.startsWith("\\\\") || /^[A-Za-z]:[\\/]/.test(path);
}

export function resolveSourcePath(sourcePath: string, sourceRoot: string): string | null {
  const normalizedPath = sourcePath.trim().replaceAll("\\", "/");
  if (!normalizedPath || normalizedPath.includes("\0")) return null;
  const segments = normalizedPath.split("/");
  if (segments.some((segment) => segment === "..")) return null;
  if (isAbsolutePath(sourcePath)) return normalizedPath;

  const normalizedRoot = sourceRoot.trim().replaceAll("\\", "/").replace(/\/$/, "");
  if (!normalizedRoot || !isAbsolutePath(normalizedRoot)) return null;
  return `${normalizedRoot}/${segments.filter((segment) => segment && segment !== ".").join("/")}`;
}

export function editorTemplate(settings: EditorSettings): string {
  if (settings.preset === "custom") return settings.customTemplate.trim();
  if (settings.preset === "jetbrains") {
    return `jetbrains://${settings.jetbrainsProduct}/navigate/reference?project={projectEncoded}&path={pathEncoded}:{line}:{column}`;
  }
  return presetTemplates[settings.preset];
}

export function validateEditorTemplate(template: string): string | null {
  const protocol = template.match(/^([a-z][a-z0-9+.-]*):\/\//i)?.[1].toLowerCase();
  if (!protocol) {
    return "Enter a URL template beginning with a protocol such as my-editor://.";
  }
  if (["javascript", "data", "file", "http", "https"].includes(protocol)) {
    return "Use a local editor protocol rather than a web or file URL.";
  }
  if (
    !template.includes("{path}")
    && !template.includes("{pathEncoded}")
    && !template.includes("{pathNoLeadingSlash}")
  ) {
    return "The URL template must contain a path placeholder.";
  }
  return null;
}

export function buildEditorURL(
  settings: EditorSettings,
  fullPath: string,
  source: SourceLocation,
  projectName = "",
): string | null {
  const template = editorTemplate(settings);
  if (validateEditorTemplate(template)) return null;
  if ((template.includes("{project}") || template.includes("{projectEncoded}")) && !projectName) return null;

  const pathForURL = fullPath
    .split("/")
    .map((segment) => encodeURIComponent(segment))
    .join("/");

  return template
    .replaceAll("{path}", pathForURL)
    .replaceAll("{pathEncoded}", encodeURIComponent(fullPath))
    .replaceAll("{pathNoLeadingSlash}", pathForURL.replace(/^\//, ""))
    .replaceAll("{project}", projectName.split("/").map((segment) => encodeURIComponent(segment)).join("/"))
    .replaceAll("{projectEncoded}", encodeURIComponent(projectName))
    .replaceAll("{line}", String(source.line))
    .replaceAll("{column}", String(source.column));
}

export function sourceLocationLabel(source: SourceLocation) {
  return `${source.path}:${source.line}${source.column > 1 ? `:${source.column}` : ""}`;
}
