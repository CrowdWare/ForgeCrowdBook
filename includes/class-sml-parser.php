<?php

if (!defined('ABSPATH')) {
    exit;
}

class SML_Parser
{
    private string $source = '';
    private int $length = 0;
    private int $pos = 0;

    public function parse(string $source): array
    {
        $this->source = $source;
        $this->length = strlen($source);
        $this->pos = 0;

        $nodes = [];
        while (true) {
            $this->skipWhitespace();
            if ($this->eof()) {
                break;
            }
            $nodes[] = $this->parseComponent();
        }

        return $nodes;
    }

    private function parseComponent(): array
    {
        $name = $this->parseIdentifier();
        $this->skipWhitespace();
        $this->expect('{');

        $props = [];
        $children = [];

        while (true) {
            $this->skipWhitespace();
            if ($this->peek() === '}') {
                $this->pos++;
                break;
            }

            $identifier = $this->parseIdentifier();
            $this->skipWhitespace();
            $next = $this->peek();

            if ($next === ':') {
                $this->pos++;
                $this->skipWhitespace();
                $props[$identifier] = $this->parseValueList();
            } elseif ($next === '{') {
                $children[] = $this->parseComponentWithKnownName($identifier);
            } else {
                throw new RuntimeException('Expected : or { after identifier ' . $identifier . ' at position ' . $this->pos);
            }
        }

        return [
            'type' => $name,
            'props' => $props,
            'children' => $children,
        ];
    }

    private function parseComponentWithKnownName(string $name): array
    {
        $this->expect('{');

        $props = [];
        $children = [];

        while (true) {
            $this->skipWhitespace();
            if ($this->peek() === '}') {
                $this->pos++;
                break;
            }

            $identifier = $this->parseIdentifier();
            $this->skipWhitespace();
            $next = $this->peek();

            if ($next === ':') {
                $this->pos++;
                $this->skipWhitespace();
                $props[$identifier] = $this->parseValueList();
            } elseif ($next === '{') {
                $children[] = $this->parseComponentWithKnownName($identifier);
            } else {
                throw new RuntimeException('Expected : or { after identifier ' . $identifier . ' at position ' . $this->pos);
            }
        }

        return [
            'type' => $name,
            'props' => $props,
            'children' => $children,
        ];
    }

    private function parseValueList(): mixed
    {
        $values = [$this->parseValue()];

        while (true) {
            $this->skipWhitespace();
            if ($this->peek() !== ',') {
                break;
            }
            $this->pos++;
            $this->skipWhitespace();
            $values[] = $this->parseValue();
        }

        return count($values) === 1 ? $values[0] : $values;
    }

    private function parseValue(): mixed
    {
        $char = $this->peek();

        if ($char === '"' || $char === "'") {
            return $this->parseString();
        }

        if ($char === '-' || ctype_digit($char)) {
            return $this->parseNumber();
        }

        $identifier = $this->parseIdentifier();
        $lower = strtolower($identifier);
        if ($lower === 'true') {
            return true;
        }
        if ($lower === 'false') {
            return false;
        }

        return $identifier;
    }

    private function parseString(): string
    {
        $quote = $this->peek();
        $this->pos++;

        $result = '';
        while (!$this->eof()) {
            $ch = $this->source[$this->pos++];
            if ($ch === '\\' && !$this->eof()) {
                $result .= $this->source[$this->pos++];
                continue;
            }
            if ($ch === $quote) {
                return $result;
            }
            $result .= $ch;
        }

        throw new RuntimeException('Unterminated string literal');
    }

    private function parseNumber(): float|int
    {
        $start = $this->pos;
        if ($this->peek() === '-') {
            $this->pos++;
        }

        while (!$this->eof() && ctype_digit($this->peek())) {
            $this->pos++;
        }

        if (!$this->eof() && $this->peek() === '.') {
            $this->pos++;
            while (!$this->eof() && ctype_digit($this->peek())) {
                $this->pos++;
            }
        }

        $raw = substr($this->source, $start, $this->pos - $start);
        return str_contains($raw, '.') ? (float) $raw : (int) $raw;
    }

    private function parseIdentifier(): string
    {
        $this->skipWhitespace();

        if ($this->eof()) {
            throw new RuntimeException('Unexpected end of source while reading identifier');
        }

        $start = $this->pos;
        $char = $this->peek();
        if (!preg_match('/[A-Za-z_]/', $char)) {
            throw new RuntimeException('Invalid identifier start at position ' . $this->pos);
        }

        $this->pos++;
        while (!$this->eof() && preg_match('/[A-Za-z0-9_\-]/', $this->peek())) {
            $this->pos++;
        }

        return substr($this->source, $start, $this->pos - $start);
    }

    private function skipWhitespace(): void
    {
        while (!$this->eof()) {
            $ch = $this->peek();
            if (ctype_space($ch)) {
                $this->pos++;
                continue;
            }

            // line comments: // ...
            if ($ch === '/' && ($this->pos + 1 < $this->length) && $this->source[$this->pos + 1] === '/') {
                $this->pos += 2;
                while (!$this->eof() && $this->peek() !== "\n") {
                    $this->pos++;
                }
                continue;
            }

            break;
        }
    }

    private function expect(string $char): void
    {
        $this->skipWhitespace();
        if ($this->peek() !== $char) {
            throw new RuntimeException('Expected "' . $char . '" at position ' . $this->pos);
        }
        $this->pos++;
    }

    private function peek(): string
    {
        return $this->source[$this->pos] ?? '';
    }

    private function eof(): bool
    {
        return $this->pos >= $this->length;
    }
}
