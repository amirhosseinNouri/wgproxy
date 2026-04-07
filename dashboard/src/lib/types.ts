export interface User {
  username: string;
  upload: number;
  download: number;
  enabled: boolean;
  created_at: string;
  online: boolean;
  connections: number;
}

export interface Stats {
  total_upload: number;
  total_download: number;
  user_count: number;
  active_users: number;
  online_users: number;
  uptime_seconds: number;
}

export interface LoginResponse {
  token: string;
  must_change_password: boolean;
}
