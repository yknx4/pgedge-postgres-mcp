/*-------------------------------------------------------------------------
 *
 * pgEdge MCP Client - SSE Chat Streaming Helper Tests
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 *-------------------------------------------------------------------------
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { sseChat } from './sseChat';

/**
 * Builds a Response-like object whose body is a ReadableStream that
 * yields the supplied byte chunks. Use this to fake fetch() returning
 * a server-sent-event stream.
 *
 * @param {string[]} chunks - Strings to emit in order; each becomes a
 *     Uint8Array enqueued onto the stream.
 * @param {object} [opts] - Optional overrides (status, statusText, ok).
 * @returns {object} - Response-like object.
 */
const buildStreamResponse = (chunks, opts = {}) => {
    const encoder = new TextEncoder();
    const stream = new ReadableStream({
        start(controller) {
            for (const chunk of chunks) {
                controller.enqueue(encoder.encode(chunk));
            }
            controller.close();
        },
    });
    return {
        ok: opts.ok ?? true,
        status: opts.status ?? 200,
        statusText: opts.statusText ?? 'OK',
        body: stream,
        text: async () => opts.text ?? '',
    };
};

describe('sseChat', () => {
    let originalFetch;

    beforeEach(() => {
        originalFetch = globalThis.fetch;
    });

    afterEach(() => {
        globalThis.fetch = originalFetch;
        vi.restoreAllMocks();
    });

    it('assembles text chunks into a single text block', async () => {
        globalThis.fetch = vi.fn().mockResolvedValue(buildStreamResponse([
            'data: {"type":"text","text":"Hello"}\n\n',
            'data: {"type":"text","text":" world"}\n\n',
            'event: done\ndata: {"stop_reason":"end_turn","usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}\n\n',
        ]));

        const seen = [];
        const result = await sseChat(
            { messages: [] },
            { onTextChunk: (c) => seen.push(c) },
        );

        expect(seen).toEqual(['Hello', ' world']);
        expect(result.stop_reason).toBe('end_turn');
        expect(result.usage).toEqual({
            prompt_tokens: 5,
            completion_tokens: 2,
            total_tokens: 7,
        });
        expect(result.content).toEqual([
            { type: 'text', text: 'Hello world' },
        ]);
    });

    it('assembles tool_use_start + tool_use_delta into a tool_use block', async () => {
        globalThis.fetch = vi.fn().mockResolvedValue(buildStreamResponse([
            'data: {"type":"text","text":"Looking up weather."}\n\n',
            'data: {"type":"tool_use_start","tool_use":{"id":"tu_1","name":"get_weather","input":null}}\n\n',
            'data: {"type":"tool_use_delta","partial":"{\\"city\\":"}\n\n',
            'data: {"type":"tool_use_delta","partial":"\\"London\\"}"}\n\n',
            'event: done\ndata: {"stop_reason":"tool_use","usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}\n\n',
        ]));

        const result = await sseChat({ messages: [] });

        expect(result.stop_reason).toBe('tool_use');
        expect(result.content).toEqual([
            { type: 'text', text: 'Looking up weather.' },
            {
                type: 'tool_use',
                tool_use: {
                    id: 'tu_1',
                    name: 'get_weather',
                    input: { city: 'London' },
                },
            },
        ]);
    });

    it('throws when the server emits an error event', async () => {
        globalThis.fetch = vi.fn().mockResolvedValue(buildStreamResponse([
            'data: {"type":"text","text":"partial"}\n\n',
            'event: error\ndata: {"error":"upstream timeout"}\n\n',
        ]));

        await expect(sseChat({ messages: [] })).rejects.toThrow('upstream timeout');
    });

    it('throws when the response is not ok', async () => {
        globalThis.fetch = vi.fn().mockResolvedValue({
            ok: false,
            status: 500,
            statusText: 'Internal Server Error',
            body: null,
            text: async () => 'boom',
        });

        await expect(sseChat({ messages: [] })).rejects.toThrow(/HTTP 500/);
    });

    it('forwards Authorization header when sessionToken is provided', async () => {
        const fetchMock = vi.fn().mockResolvedValue(buildStreamResponse([
            'event: done\ndata: {"stop_reason":"end_turn"}\n\n',
        ]));
        globalThis.fetch = fetchMock;

        await sseChat({ messages: [] }, { sessionToken: 'abc123' });

        expect(fetchMock).toHaveBeenCalledTimes(1);
        const [url, init] = fetchMock.mock.calls[0];
        expect(url).toBe('/api/llm/v1/chat/stream');
        expect(init.method).toBe('POST');
        expect(init.headers['Authorization']).toBe('Bearer abc123');
        expect(init.headers['Accept']).toBe('text/event-stream');
        expect(init.headers['Content-Type']).toBe('application/json');
    });

    it('handles frames split across multiple network chunks', async () => {
        globalThis.fetch = vi.fn().mockResolvedValue(buildStreamResponse([
            'data: {"type":"text","text":"He',
            'llo"}\n\ndata: {"type":"text","text":" world"}\n',
            '\nevent: done\ndata: {"stop_reason":"end_turn"}\n\n',
        ]));

        const result = await sseChat({ messages: [] });
        expect(result.content).toEqual([{ type: 'text', text: 'Hello world' }]);
        expect(result.stop_reason).toBe('end_turn');
    });

    it('prefers content array from done payload when present', async () => {
        globalThis.fetch = vi.fn().mockResolvedValue(buildStreamResponse([
            'data: {"type":"text","text":"streamed"}\n\n',
            'event: done\ndata: {"stop_reason":"end_turn","content":[{"type":"text","text":"final"}]}\n\n',
        ]));

        const result = await sseChat({ messages: [] });
        expect(result.content).toEqual([{ type: 'text', text: 'final' }]);
    });
});
