/*-------------------------------------------------------------------------
 *
 * pgEdge MCP Client - useLLMProviders Hook Tests
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 *-------------------------------------------------------------------------
 */

import { describe, it, expect } from 'vitest';
import { normaliseProviders } from '../useLLMProviders';

describe('normaliseProviders', () => {
    it('prefers the API display_name when present', () => {
        const out = normaliseProviders([
            { name: 'anthropic', display_name: 'Anthropic', model: 'claude' },
        ]);
        expect(out[0].display).toBe('Anthropic');
    });

    it('falls back to the local label map when display_name is absent', () => {
        const out = normaliseProviders([{ name: 'anthropic', model: 'claude' }]);
        expect(out[0].display).toBe('Anthropic Claude');
    });

    it('capitalises unknown providers with no display_name', () => {
        const out = normaliseProviders([{ name: 'weird', model: 'x' }]);
        expect(out[0].display).toBe('Weird');
    });
});
