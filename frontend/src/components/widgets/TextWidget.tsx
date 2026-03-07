import type { Widget } from "../../types/dashboard";

interface Props {
  widget: Widget;
}

export function TextWidget({ widget }: Props) {
  if (!widget.content) {
    return null;
  }

  return (
    <div
      className="prose prose-sm max-w-none text-[var(--dac-text-primary)]"
      dangerouslySetInnerHTML={{ __html: renderMarkdown(widget.content) }}
    />
  );
}

function renderMarkdown(text: string): string {
  const lines = text.split("\n");
  const html: string[] = [];
  let inUl = false;
  let inOl = false;
  let inBlockquote = false;

  const flushList = () => {
    if (inUl) { html.push("</ul>"); inUl = false; }
    if (inOl) { html.push("</ol>"); inOl = false; }
  };

  const flushBlockquote = () => {
    if (inBlockquote) { html.push("</blockquote>"); inBlockquote = false; }
  };

  for (const line of lines) {
    // Horizontal rule
    if (/^(-{3,}|\*{3,}|_{3,})\s*$/.test(line)) {
      flushList();
      flushBlockquote();
      html.push("<hr />");
      continue;
    }

    // Headers
    const headerMatch = line.match(/^(#{1,6})\s+(.+)$/);
    if (headerMatch) {
      flushList();
      flushBlockquote();
      const level = headerMatch[1].length;
      html.push(`<h${level}>${inlineFormat(headerMatch[2])}</h${level}>`);
      continue;
    }

    // Blockquote
    if (line.startsWith("> ")) {
      flushList();
      if (!inBlockquote) { html.push("<blockquote>"); inBlockquote = true; }
      html.push(`<p>${inlineFormat(line.slice(2))}</p>`);
      continue;
    } else if (inBlockquote) {
      flushBlockquote();
    }

    // Unordered list
    if (/^[-*]\s+/.test(line)) {
      if (inOl) { html.push("</ol>"); inOl = false; }
      if (!inUl) { html.push("<ul>"); inUl = true; }
      html.push(`<li>${inlineFormat(line.replace(/^[-*]\s+/, ""))}</li>`);
      continue;
    }

    // Ordered list
    const olMatch = line.match(/^(\d+)[.)]\s+(.+)$/);
    if (olMatch) {
      if (inUl) { html.push("</ul>"); inUl = false; }
      if (!inOl) { html.push("<ol>"); inOl = true; }
      html.push(`<li>${inlineFormat(olMatch[2])}</li>`);
      continue;
    }

    flushList();

    // Empty line
    if (line.trim() === "") {
      html.push("");
      continue;
    }

    // Paragraph
    html.push(`<p>${inlineFormat(line)}</p>`);
  }

  flushList();
  flushBlockquote();

  return html.join("\n");
}

function inlineFormat(text: string): string {
  return (
    text
      // Images: ![alt](src)
      .replace(/!\[([^\]]*)\]\(([^)]+)\)/g, '<img src="$2" alt="$1" style="max-width:100%;border-radius:4px" />')
      // Links: [text](url)
      .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer" style="color:var(--dac-accent);text-decoration:underline">$1</a>')
      // Bold + italic: ***text*** or ___text___
      .replace(/\*{3}(.+?)\*{3}/g, "<strong><em>$1</em></strong>")
      .replace(/_{3}(.+?)_{3}/g, "<strong><em>$1</em></strong>")
      // Bold: **text** or __text__
      .replace(/\*{2}(.+?)\*{2}/g, "<strong>$1</strong>")
      .replace(/_{2}(.+?)_{2}/g, "<strong>$1</strong>")
      // Italic: *text* or _text_
      .replace(/\*(.+?)\*/g, "<em>$1</em>")
      .replace(/(?<!\w)_(.+?)_(?!\w)/g, "<em>$1</em>")
      // Strikethrough: ~~text~~
      .replace(/~~(.+?)~~/g, "<del>$1</del>")
      // Inline code: `code`
      .replace(/`(.+?)`/g, '<code style="background:var(--dac-surface-hover,#f3f4f6);padding:1px 4px;border-radius:3px;font-size:0.875em">$1</code>')
  );
}
