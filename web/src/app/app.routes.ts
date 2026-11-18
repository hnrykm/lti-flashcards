import { Routes } from '@angular/router';
import { DeckList } from './deck-list';
import { DeckView } from './deck-view';
import { Study } from './study';

export const routes: Routes = [
  { path: '', component: DeckList, title: 'Decks · Flashcards' },
  { path: 'decks/:id', component: DeckView, title: 'Deck · Flashcards' },
  { path: 'decks/:id/study', component: Study, title: 'Study · Flashcards' },
  { path: '**', redirectTo: '' },
];
