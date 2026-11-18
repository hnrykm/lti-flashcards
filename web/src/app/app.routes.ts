import { Routes } from '@angular/router';
import { DeckList } from './deck-list';

export const routes: Routes = [
  { path: '', component: DeckList, title: 'Decks · Flashcards' },
  { path: '**', redirectTo: '' },
];
