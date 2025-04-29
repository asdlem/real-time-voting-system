export interface PollOption {
  id?: number;
  ID?: number;
  text?: string;
  option_text?: string;
  votes?: number;
  poll_id?: number;
  CreatedAt?: string;
  UpdatedAt?: string;
  DeletedAt?: string | null;
}

export interface Poll {
  id?: number;
  ID?: number;
  title?: string;
  question?: string;
  description?: string;
  active?: boolean;
  is_active?: boolean;
  poll_type?: number;
  type?: number;
  options: PollOption[];
  created_at?: string;
  updated_at?: string;
  results?: PollOptionResult[];
  CreatedAt?: string;
  UpdatedAt?: string;
  DeletedAt?: string | null;
  end_time?: string;
}

export interface PollVoteRequest {
  option_id?: number;
  option_ids?: number[];
}

export interface PollOptionResult {
  id?: number;
  option_id?: number;
  text?: string;
  votes: number;
  percentage: number;
}

export interface PollVoteResponse {
  success?: boolean;
  message?: string;
  results?: PollOptionResult[];
  current_results?: PollOptionResult[];
}

export interface CreatePollRequest {
  title?: string;
  question: string;
  description?: string;
  options: { text: string }[];
  poll_type: number;
  active: boolean;
  min_options?: number;
  max_options?: number;
  end_time?: string;
} 