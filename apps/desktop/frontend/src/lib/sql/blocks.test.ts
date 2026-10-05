import { describe, expect, it } from 'vitest';
import cases from '../../../e2e/split-cases.json';
import features from '../../../e2e/engines.json';
import type { EngineFeatures } from '../api/wire';
import { statementRanges } from './statements';

describe('splitting around BEGIN … END and DELIMITER', () => {
  for (const tc of cases) {
    it(tc.name, () => {
      const syntax = (features as Record<string, EngineFeatures>)[tc.engine].syntax;
      expect(statementRanges(tc.in, syntax).map(r => r.text)).toEqual(tc.want);
    });
  }
});
