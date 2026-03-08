import { Database } from '../db';

export interface User {
  id: string;
  name: string;
  email: string;
}

export class UserService {
  private db: Database;

  constructor() {
    this.db = new Database();
  }

  async getUser(id: string): Promise<User | null> {
    return this.db.findById('users', id);
  }

  async createUser(user: Omit<User, 'id'>): Promise<User> {
    return this.db.insert('users', user);
  }
}
