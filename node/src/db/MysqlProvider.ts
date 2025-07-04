import {
  Pool,
  PoolConnection,
  ResultSetHeader,
  RowDataPacket,
  createPool,
} from "mysql2/promise";

export interface DatabaseConfig {
  host: string;
  user: string;
  password: string;
  database: string;
  port?: number;
  connectionLimit?: number;
}

export class DatabaseProvider<T extends RowDataPacket = any> {
  private pool: Pool;

  constructor(config: DatabaseConfig) {
    this.pool = createPool({
      host: config.host,
      user: config.user,
      password: config.password,
      database: config.database,
      port: config.port || 3306,
      connectionLimit: config.connectionLimit || 10,
      waitForConnections: true,
      queueLimit: 0,
    });
  }

  async getConnection(): Promise<PoolConnection> {
    try {
      return await this.pool.getConnection();
    } catch (error) {
      throw new Error(`Failed to get database connection: ${error}`);
    }
  }

  async query<R>(query: string, params: any[] = []): Promise<R> {
    let connection: PoolConnection | null = null;
    try {
      connection = await this.getConnection();
      const [results] = await connection.query(query, params);
      return results as R;
    } catch (error) {
      throw new Error(`Query execution failed: ${error}`);
    } finally {
      if (connection) {
        connection.release();
      }
    }
  }

  async insert(table: string, data: Partial<T>): Promise<number> {
    const columns = Object.keys(data).join(", ");
    const placeholders = Object.keys(data)
      .map(() => "?")
      .join(", ");
    const values = Object.values(data);

    const query = `INSERT INTO ${table} (${columns}) VALUES (${placeholders})`;
    const result = await this.query<ResultSetHeader>(query, values);
    return result.insertId;
  }

  async insertBulk(table: string, data: Partial<T>[]): Promise<number[]> {
    if (!data.length) return [];

    const columns = Object.keys(data[0]).join(", ");
    const placeholders = data
      .map(
        () =>
          `(${Object.keys(data[0])
            .map(() => "?")
            .join(", ")})`
      )
      .join(", ");
    const values = data.flatMap((item) => Object.values(item));

    const query = `INSERT INTO ${table} (${columns}) VALUES ${placeholders}`;
    const result = await this.query<ResultSetHeader>(query, values);
    return Array.from(
      { length: result.affectedRows },
      (_, i) => result.insertId + i
    );
  }

  async update(
    table: string,
    data: Partial<T>,
    where: Partial<T> | string
  ): Promise<number> {
    const setClause = Object.keys(data)
      .map((key) => `${key} = ?`)
      .join(", ");
    const values = Object.values(data);

    let whereClause = "";
    let whereValues: any[] = [];

    if (typeof where === "string") {
      whereClause = where;
    } else {
      whereClause = Object.keys(where)
        .map((key) => `${key} = ?`)
        .join(" AND ");
      whereValues = Object.values(where);
    }

    const query = `UPDATE ${table} SET ${setClause} WHERE ${whereClause}`;
    const result = await this.query<ResultSetHeader>(query, [
      ...values,
      ...whereValues,
    ]);
    return result.affectedRows;
  }

  async select<K extends keyof T>(
    table: string,
    where?: Partial<T> | string,
    columns: K[] = ["*"] as K[]
  ): Promise<T[]> {
    const selectColumns = columns.join(", ");
    let query = `SELECT ${selectColumns} FROM ${table}`;
    let values: any[] = [];

    if (where) {
      if (typeof where === "string") {
        query += ` WHERE ${where}`;
      } else {
        query += ` WHERE ${Object.keys(where)
          .map((key) => `${key} = ?`)
          .join(" AND ")}`;
        values = Object.values(where);
      }
    }

    return this.query<T[]>(query, values);
  }

  async delete(table: string, where: Partial<T> | string): Promise<number> {
    let whereClause = "";
    let values: any[] = [];

    if (typeof where === "string") {
      whereClause = where;
    } else {
      whereClause = Object.keys(where)
        .map((key) => `${key} = ?`)
        .join(" AND ");
      values = Object.values(where);
    }

    const query = `DELETE FROM ${table} WHERE ${whereClause}`;
    const result = await this.query<ResultSetHeader>(query, values);
    return result.affectedRows;
  }

  async close(): Promise<void> {
    await this.pool.end();
  }
}

const dbConfig: DatabaseConfig = {
  host: process.env.DB_HOST!,
  user: process.env.DB_USER!,
  password: process.env.DB_PASSWORD!,
  database: process.env.DB_DATABASE!,
  port: process.env.DB_PORT ? parseInt(process.env.DB_PORT, 10) : undefined,
};

export const db = new DatabaseProvider(dbConfig);
