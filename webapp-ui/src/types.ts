export type FeatureFlagValue = 'disabled' | 'default' | 'forced' | string;

export interface ServerConfig {
  title?: string;
  abuseContact?: string;
  authentication?: boolean;
  googleAuthentication?: boolean;
  ovhAuthentication?: boolean;
  [key: `feature_${string}`]: FeatureFlagValue | any;
  maxFileSize?: number;
  ttl?: number;
}

export interface UserInfo {
  id: string;
  login: string;
  admin?: boolean;
  maxFileSize?: number;
}

export interface UploadDescription {
  id: string;
  uploadToken?: string;
  removable?: boolean;
  admin?: boolean;
  stream?: boolean;
  oneShot?: boolean;
  comments?: string;
  ttl?: number;
  files?: Array<{ id: string; fileName: string; size: number }>; 
}

export interface ClientBinary {
  name: string;
  md5?: string;
  path: string;
  showDetails?: boolean;
}
