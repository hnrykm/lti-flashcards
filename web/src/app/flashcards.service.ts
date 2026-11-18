import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';

export interface Deck {
  id: number;
  course_id: number;
  name: string;
  created_at: string;
}

export interface Card {
  id: number;
  deck_id: number;
  front: string;
  back: string;
  created_at: string;
}

export interface CardInput {
  front: string;
  back: string;
}

/** Wraps the flashcards API. The interceptor adds the bearer token. */
@Injectable({ providedIn: 'root' })
export class FlashcardsService {
  private readonly http = inject(HttpClient);

  listDecks(): Observable<Deck[]> {
    return this.http.get<Deck[]>('/api/decks');
  }

  getDeck(id: number): Observable<Deck> {
    return this.http.get<Deck>(`/api/decks/${id}`);
  }

  createDeck(name: string): Observable<Deck> {
    return this.http.post<Deck>('/api/decks', { name });
  }

  updateDeck(id: number, name: string): Observable<Deck> {
    return this.http.put<Deck>(`/api/decks/${id}`, { name });
  }

  deleteDeck(id: number): Observable<void> {
    return this.http.delete<void>(`/api/decks/${id}`);
  }

  listCards(deckId: number): Observable<Card[]> {
    return this.http.get<Card[]>(`/api/decks/${deckId}/cards`);
  }

  createCard(deckId: number, card: CardInput): Observable<Card> {
    return this.http.post<Card>(`/api/decks/${deckId}/cards`, card);
  }

  updateCard(id: number, card: CardInput): Observable<Card> {
    return this.http.put<Card>(`/api/cards/${id}`, card);
  }

  deleteCard(id: number): Observable<void> {
    return this.http.delete<void>(`/api/cards/${id}`);
  }
}

/** Turns an HTTP error into a sentence for the error alert. */
export function errorMessage(err: unknown): string {
  if (err instanceof HttpErrorResponse) {
    if (err.status === 0) return 'Could not reach the server. Check your connection and try again.';
    if (err.status === 401) return 'Your session has expired. Open the tool from your course again.';
    if (err.status === 403) return 'Only instructors can make changes.';
    if (err.status === 404) return 'That deck or card no longer exists.';
    if (typeof err.error?.error === 'string') return err.error.error;
  }
  return 'Something went wrong. Please try again.';
}
