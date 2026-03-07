import type { Widget } from "../../types/dashboard";

interface Props {
  widget: Widget;
}

export function TextWidget({ widget }: Props) {
  if (!widget.content) {
    return null;
  }

  // Simple markdown-ish rendering: bold, headers, lists.
  // For full markdown, we'd add a dependency like react-markdown.
  return (
    <div
      className="prose prose-sm max-w-none text-[var(--dac-text-primary)]"
      dangerouslySetInnerHTML={{ __html: simpleMarkdown(widget.content) }}
    />
  );
}

function simpleMarkdown(text: string): string {
  return text
    .split("\n")
    .map((line) => {
      // Headers
      if (line.startsWith("### ")) return `<h3>${line.slice(4)}</h3>`;
      if (line.startsWith("## ")) return `<h2>${line.slice(3)}</h2>`;
      if (line.startsWith("# ")) return `<h1>${line.slice(2)}</h1>`;
      // List items
      if (line.startsWith("- ")) return `<li>${inlineFormat(line.slice(2))}</li>`;
      // Paragraphs
      if (line.trim() === "") return "";
      return `<p>${inlineFormat(line)}</p>`;
    })
    .join("\n");
}

function inlineFormat(text: string): string {
  return text.replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
             .replace(/`(.+?)`/g, "<code>$1</code>");
}
