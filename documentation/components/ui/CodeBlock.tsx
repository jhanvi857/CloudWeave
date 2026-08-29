'use client'

import React, { useState, useMemo } from 'react'
import { Check, Copy, Terminal, Code2 } from 'lucide-react'

interface CodeBlockProps {
  code: string
  language?: string
  filename?: string
  showLineNumbers?: boolean
}

// Token types for colorful syntax highlighting
interface Token {
  text: string
  type: 'keyword' | 'type' | 'function' | 'string' | 'comment' | 'number' | 'flag' | 'operator' | 'plain'
}

function tokenizeLine(line: string, lang: string): Token[] {
  const tokens: Token[] = []
  let remaining = line

  // Comment check first
  const commentPrefix = (lang === 'bash' || lang === 'sh' || lang === 'python' || lang === 'py') ? '#' : '//'
  const commentIdx = remaining.indexOf(commentPrefix)

  let beforeComment = remaining
  let commentStr = ''

  if (commentIdx !== -1) {
    // Check that comment prefix is not inside a string
    const stringMatches = remaining.match(/(["'`])(?:(?=(\\?))\2.)*?\1/g) || []
    let isInsideString = false
    let currentPos = 0
    for (const sm of stringMatches) {
      const matchPos = remaining.indexOf(sm, currentPos)
      if (commentIdx > matchPos && commentIdx < matchPos + sm.length) {
        isInsideString = true
        break
      }
      currentPos = matchPos + sm.length
    }

    if (!isInsideString) {
      beforeComment = remaining.slice(0, commentIdx)
      commentStr = remaining.slice(commentIdx)
    }
  }

  // Tokenize the code part
  if (beforeComment.length > 0) {
    // Regex for matching tokens in code
    const tokenRegex = /(["'`])(?:(?=(\\?))\2.)*?\1|--?[a-zA-Z0-9_\-]+|\b0x[0-9a-fA-F]+\b|\b\d+(?:\.\d+)?(?:ms|s|ns|MB|GB|KB|%)?\b|\b(?:func|package|import|type|struct|interface|return|if|else|for|range|var|const|make|new|len|append|panic|defer|go|select|case|break|continue|default|map|chan|nil|true|false|def|class|with|as|from|in|is|not|and|or|docker|curl|aws|git|cd|mkdir|cweave|export|let|const|function|interface)\b|\b(?:string|int|int64|uint|uint32|uint64|uint8|byte|bool|error|any|float64|float32|Chunk|ChunkID|Node|Manifest|WAL|WALEntry|DiskStore|HashRing|CASStore|Handler|Config|Context|Response|Request|ResponseWriter|VectorClock|ShardBlock|IndexEntry|Index|Header|State)\b|\b[a-zA-Z_][a-zA-Z0-9_]*(?=\s*\()|[:=+\-*\/&|<>\^~%!,;.\(\)\[\]\{\}]|[^\s:=+\-*\/&|<>\^~%!,;.\(\)\[\]\{\}]+|\s+/g

    let match: RegExpExecArray | null
    while ((match = tokenRegex.exec(beforeComment)) !== null) {
      const text = match[0]

      // String literal
      if (text.startsWith('"') || text.startsWith("'") || text.startsWith('`')) {
        tokens.push({ text, type: 'string' })
      }
      // CLI Flag
      else if (text.startsWith('-')) {
        tokens.push({ text, type: 'flag' })
      }
      // Hex or numbers
      else if (/^\b0x[0-9a-fA-F]+\b$/.test(text) || /^\b\d+(?:\.\d+)?(?:ms|s|ns|MB|GB|KB|%)?\b$/.test(text)) {
        tokens.push({ text, type: 'number' })
      }
      // Keywords
      else if (/^(?:func|package|import|type|struct|interface|return|if|else|for|range|var|const|make|new|len|append|panic|defer|go|select|case|break|continue|default|map|chan|nil|true|false|def|class|with|as|from|in|is|not|and|or|docker|curl|aws|git|cd|mkdir|cweave|export|let|const|function)$/.test(text)) {
        tokens.push({ text, type: 'keyword' })
      }
      // Types
      else if (/^(?:string|int|int64|uint|uint32|uint64|uint8|byte|bool|error|any|float64|float32|Chunk|ChunkID|Node|Manifest|WAL|WALEntry|DiskStore|HashRing|CASStore|Handler|Config|Context|Response|Request|ResponseWriter|VectorClock|ShardBlock|IndexEntry|Index|Header|State)$/.test(text)) {
        tokens.push({ text, type: 'type' })
      }
      // Function call
      else if (/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(text) && beforeComment[tokenRegex.lastIndex] === '(') {
        tokens.push({ text, type: 'function' })
      }
      // Operator / Punctuation
      else if (/^[:=+\-*\/&|<>\^~%!,;.\(\)\[\]\{\}]$/.test(text)) {
        tokens.push({ text, type: 'operator' })
      }
      // Plain
      else {
        tokens.push({ text, type: 'plain' })
      }
    }
  }

  // Append comment if exists
  if (commentStr.length > 0) {
    tokens.push({ text: commentStr, type: 'comment' })
  }

  return tokens
}

function TokenSpan({ token }: { token: Token }) {
  switch (token.type) {
    case 'keyword':
      // Vibrant Magenta/Purple for keywords
      return <span className="text-[#F472B6] font-semibold">{token.text}</span>
    case 'type':
      // Vibrant Sky/Cyan for types & structs
      return <span className="text-[#38BDF8] font-semibold">{token.text}</span>
    case 'function':
      // Vibrant Warm Yellow/Gold for functions & methods
      return <span className="text-[#FACC15]">{token.text}</span>
    case 'string':
      // Vibrant Emerald Green for strings
      return <span className="text-[#4ADE80]">{token.text}</span>
    case 'comment':
      // Elegant Muted Slate for comments
      return <span className="text-[#8B8B8F] italic">{token.text}</span>
    case 'number':
      // Vibrant Orange/Amber for numbers & hex
      return <span className="text-[#FB923C] font-semibold">{token.text}</span>
    case 'flag':
      // Vibrant Coral/Rose for CLI flags
      return <span className="text-[#FB7185]">{token.text}</span>
    case 'operator':
      // Steel for punctuation & operators
      return <span className="text-[#A6A9AE]">{token.text}</span>
    case 'plain':
    default:
      return <span className="text-[#EDE8DF]">{token.text}</span>
  }
}

export function CodeBlock({
  code,
  language = 'bash',
  filename = 'cloudweave',
  showLineNumbers = false,
}: CodeBlockProps) {
  const [copied, setCopied] = useState(false)

  const handleCopy = () => {
    navigator.clipboard.writeText(code)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const lines = useMemo(() => {
    return code.trim().split('\n')
  }, [code])

  const tokenizedLines = useMemo(() => {
    return lines.map((line) => tokenizeLine(line, language))
  }, [lines, language])

  return (
    <div className="my-4 rounded-xl border border-[#26262B] bg-[#19191C] overflow-hidden shadow-lg relative font-sans">
      {/* Terminal Window Top Bar */}
      <div className="flex items-center justify-between px-4 py-2.5 border-b border-[#26262B] bg-[#000000]">
        <div className="flex items-center space-x-2">
          <div className="w-2.5 h-2.5 rounded-full bg-[#7A1F2B]" />
          <div className="w-2.5 h-2.5 rounded-full bg-[#26262B]" />
          <div className="w-2.5 h-2.5 rounded-full bg-[#34343A]" />
        </div>

        <div className="flex items-center gap-1.5 font-mono text-xs text-[#A6A9AE]">
          {language === 'go' ? (
            <Code2 className="w-3.5 h-3.5 text-[#38BDF8]" />
          ) : (
            <Terminal className="w-3.5 h-3.5 text-[#F472B6]" />
          )}
          <span className="text-[#EDE8DF] font-semibold">{filename}</span>
          <span className="text-[#8B8B8F]">({language})</span>
        </div>

        <button
          onClick={handleCopy}
          className="flex items-center justify-center h-6 px-2.5 rounded bg-[#19191C] hover:bg-[#7A1F2B] text-[#EDE8DF] text-xs transition-colors border border-[#26262B] cursor-pointer"
          title="Copy code"
        >
          {copied ? (
            <div className="flex items-center gap-1 text-[#4ADE80] font-bold">
              <Check className="w-3 h-3 text-[#4ADE80]" />
              <span>Copied</span>
            </div>
          ) : (
            <div className="flex items-center gap-1 text-[#8B8B8F] hover:text-[#EDE8DF]">
              <Copy className="w-3 h-3" />
              <span>Copy</span>
            </div>
          )}
        </button>
      </div>

      {/* Code Editor Body with Syntax Tokens */}
      <div className="p-4 font-mono text-xs leading-relaxed overflow-x-auto bg-[#000000]">
        <pre className="w-full">
          <code>
            {tokenizedLines.map((tokens, lineIdx) => (
              <div key={lineIdx} className="table-row">
                {showLineNumbers && (
                  <span className="table-cell select-none pr-4 text-right text-[#8B8B8F]/60 border-r border-[#26262B]">
                    {lineIdx + 1}
                  </span>
                )}
                <span className={`table-cell whitespace-pre ${showLineNumbers ? 'pl-4' : ''}`}>
                  {tokens.map((token, tokIdx) => (
                    <TokenSpan key={tokIdx} token={token} />
                  ))}
                  {tokens.length === 0 && ' '}
                </span>
              </div>
            ))}
          </code>
        </pre>
      </div>
    </div>
  )
}
