import { Routes } from '@angular/router';
import { DeckList } from './deck-list';
import { DeckView } from './deck-view';

export const routes: Routes = [
  { path: '', component: DeckList, title: 'Decks · Flashcards' },
  { path: 'decks/:id', component: DeckView, title: 'Deck · Flashcards' },
  { path: '**', redirectTo: '' },
];
