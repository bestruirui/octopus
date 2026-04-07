export interface BootstrapStatusResponse {
  initialized: boolean;
  message: string;
}

export interface BootstrapCreateAdminRequest {
  username: string;
  password: string;
}
