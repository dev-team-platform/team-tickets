import { Permission, RoleName } from '../enums/uam';
import { UserGender } from '../enums/users';

export interface CurrentUser {
  id: string;
  employeeCode: string;
  username: string;
  email: string;
  firstName: string;
  lastName: string | null;
  displayName: string;
  avatarUrl: string | null;
  gender: UserGender;
  birthDate: string | null;
  lastLoginAt: string | null;
  lastLogoutAt: string | null;
  roleName: RoleName;
  permissions: Permission[];
}

export interface LogoutRequest {
  returnUrl: string;
}

export interface LogoutResponse {
  logoutUrl: string;
}
