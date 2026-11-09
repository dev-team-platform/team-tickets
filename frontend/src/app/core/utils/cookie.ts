export function getCookie(name: string): string | null {
  const cookie = document.cookie.split('; ').find((item) => item.startsWith(`${name}=`));

  if (!cookie) {
    return null;
  }

  return decodeURIComponent(cookie.substring(name.length + 1));
}
