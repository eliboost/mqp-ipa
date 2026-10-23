export class AMQMessage {
  address?: string;
  messageID?: string;
  type?: string;
  text?: string;
  bodyPreview?: number[];
  properties?: StringProperties;
  headers?: HeaderProperties;
}

export enum AMQPropertyType {
  INT,
  LONG,
  BYTE,
  STRING,
  BOOLEAN,
  SHORT,
  DOUBLE,
  FLOAT,
}

export class AMQMessageProperty {
  value?: string;
  type?: AMQPropertyType;
}

export type HeaderProperties = Record<string, string>;

export type StringProperties = Record<string, AMQMessageProperty>;
