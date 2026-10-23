export class AMQQueue {
  name?: string;
  consumerCount?: number;
  messageCount?: number;
  paused?: boolean;
  enabled?: boolean;
  replica?: number;
}
