import { environment } from '../../../environments/environment';

export function buildFullEmail(firstPart: string): string {
  return `${firstPart}@${environment.emailDomain}`;
}
