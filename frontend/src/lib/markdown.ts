// A minimal, hand-rolled renderer for the conversation preview — NOT general
// markdown. It recognizes only what transcript text actually carries: fenced
// ```code``` blocks and inline `code` spans. Everything else is plain text with
// newlines preserved by the consumer. No dependency, so no supply-chain surface.

// A fence block is a ```…``` code section; an inline run is a line of text whose
// `code` spans are split out. The component renders blocks as <pre> and inline
// runs with <code> spans, preserving the surrounding whitespace.
export type Block =
  | { kind: "code"; lang: string; text: string }
  | { kind: "text"; spans: Span[] };

export interface Span {
  code: boolean;
  text: string;
}

// parseMarkdown splits a message into fenced-code blocks and text blocks. Fences
// are lines that start with ``` (a leading-whitespace-tolerant match); an
// unterminated fence runs to the end of the message.
export function parseMarkdown(input: string): Block[] {
  const lines = input.split("\n");
  const blocks: Block[] = [];
  let textBuf: string[] = [];

  const flushText = () => {
    if (textBuf.length) {
      blocks.push({ kind: "text", spans: parseInline(textBuf.join("\n")) });
      textBuf = [];
    }
  };

  for (let i = 0; i < lines.length; i++) {
    const fence = lines[i]!.match(/^\s*```(.*)$/);
    if (!fence) {
      textBuf.push(lines[i]!);
      continue;
    }
    flushText();
    const lang = fence[1]!.trim();
    const code: string[] = [];
    i++;
    while (i < lines.length && !/^\s*```\s*$/.test(lines[i]!)) {
      code.push(lines[i]!);
      i++;
    }
    // i now sits on the closing fence (or past the end for an unterminated one).
    blocks.push({ kind: "code", lang, text: code.join("\n") });
  }
  flushText();
  return blocks;
}

// parseInline splits a run of text into alternating plain and `code` spans. An
// unmatched backtick is treated as literal text (no span opened).
export function parseInline(text: string): Span[] {
  const spans: Span[] = [];
  let i = 0;
  let plain = "";
  while (i < text.length) {
    if (text[i] === "`") {
      const close = text.indexOf("`", i + 1);
      if (close > i) {
        if (plain) {
          spans.push({ code: false, text: plain });
          plain = "";
        }
        spans.push({ code: true, text: text.slice(i + 1, close) });
        i = close + 1;
        continue;
      }
    }
    plain += text[i];
    i++;
  }
  if (plain) spans.push({ code: false, text: plain });
  return spans;
}
