import { UserService } from './services/user';
import { logger } from './utils/logger';

export interface AppConfig {
  port: number;
  host: string;
}

export class App {
  private config: AppConfig;
  private userService: UserService;

  constructor(config: AppConfig) {
    this.config = config;
    this.userService = new UserService();
  }

  start(): void {
    logger.info(`Starting on ${this.config.host}:${this.config.port}`);
  }
}

export function createApp(config: AppConfig): App {
  return new App(config);
}

const defaultPort = 3000;
export default defaultPort;
