import { describe, expect, it } from 'vitest';
import { AppError, toError } from './backend';

describe('errors from the Go side', () => {
  it('turns a mapped error into an AppError with its code and the original text', () => {
    const err = toError({ code: 'refused', message: 'Nothing answers at db:5432.', detail: 'dial tcp db:5432: connect: connection refused' });
    expect(err).toBeInstanceOf(AppError);
    expect(err.message).toBe('Nothing answers at db:5432.');
    expect((err as AppError).code).toBe('refused');
    expect((err as AppError).detail).toContain('connection refused');
  });

  it('keeps plain strings and errors as they are', () => {
    expect(toError('boom').message).toBe('boom');
    const e = new Error('x');
    expect(toError(e)).toBe(e);
  });
});
