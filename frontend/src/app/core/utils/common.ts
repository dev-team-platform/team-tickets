import { HttpErrorResponse } from '@angular/common/http';
import {
  DEFAULT_DUPLICATE_ERROR_MESSAGE,
  DEFAULT_ERROR_MESSAGE,
  DEFAULT_TOO_MANY_REQUESTS_ERROR_MESSAGE,
} from '../constants/common';
import { ErrorResponse } from '../models/common';

export function isNotEmpty(value: unknown): boolean {
  if (typeof value === 'string') {
    return value.trim().length > 0;
  }

  return value !== null && value !== undefined;
}

type ErrorMappingCallback = (httpError: HttpErrorResponse) => string | null;

function formatFieldName(field: unknown): string {
  if (typeof field !== 'string') return '';

  const words = field
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .replace(/([A-Z]+)([A-Z][a-z])/g, '$1 $2')
    .trim();

  return words ? words[0].toUpperCase() + words.slice(1) : words;
}

export function resolveErrorMessage(
  error: unknown,
  mappingCallback: ErrorMappingCallback | null = null,
): string {
  if (!(error instanceof HttpErrorResponse)) {
    return DEFAULT_ERROR_MESSAGE;
  }

  if (error.status === 429) {
    return DEFAULT_TOO_MANY_REQUESTS_ERROR_MESSAGE;
  }

  if (error.status === 409) {
    const response = error.error as ErrorResponse | null;
    if (response?.message === DEFAULT_DUPLICATE_ERROR_MESSAGE) {
      const field = formatFieldName(response.details?.['field']);
      return `${field} is duplicated with value "${response.details?.['value']}"`;
    }
  }

  if (error.status >= 500 || error.status === 0) {
    return DEFAULT_ERROR_MESSAGE;
  }

  if (mappingCallback != null) {
    const message = mappingCallback(error);

    if (message != null) {
      return message;
    }
  }

  if (error.status >= 400 && error.status < 500) {
    const response = error.error as Partial<ErrorResponse> | null;
    if (typeof response?.message === 'string' && response.message.trim()) {
      return response.message.trim();
    }
  }

  return DEFAULT_ERROR_MESSAGE;
}

export function resolveTimeZoneId(): string {
  return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
}
