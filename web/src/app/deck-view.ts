import { Component, OnInit, effect, inject, input, signal } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { forkJoin } from 'rxjs';
import { Card, Deck, FlashcardsService, errorMessage } from './flashcards.service';
import { SessionService } from './session';

/** One deck's cards. Instructors can rename the deck and add, edit and delete cards. */
@Component({
  selector: 'app-deck-view',
  imports: [FormsModule, RouterLink],
  template: `
    <nav aria-label="breadcrumb">
      <ol class="breadcrumb">
        <li class="breadcrumb-item"><a routerLink="/">Decks</a></li>
        <li class="breadcrumb-item active" aria-current="page">{{ deck()?.name ?? 'Deck' }}</li>
      </ol>
    </nav>

    @if (error()) {
      <div class="alert alert-danger" role="alert">{{ error() }}</div>
    }

    @if (loading()) {
      <div class="d-flex align-items-center gap-2" role="status">
        <div class="spinner-border spinner-border-sm" aria-hidden="true"></div>
        <span>Loading cards…</span>
      </div>
    } @else if (deck(); as d) {
      @if (renaming()) {
        <form class="row g-2 align-items-end mb-3" (ngSubmit)="rename()">
          <div class="col-sm-8 col-md-6">
            <label for="deck-name" class="form-label">Deck name</label>
            <input id="deck-name" name="name" class="form-control" maxlength="100" required
                   [(ngModel)]="deckName" />
          </div>
          <div class="col-auto d-flex gap-2">
            <button type="submit" class="btn btn-primary" [disabled]="!deckName().trim()">Save</button>
            <button type="button" class="btn btn-outline-secondary" (click)="renaming.set(false)">Cancel</button>
          </div>
        </form>
      } @else {
        <div class="d-flex flex-wrap align-items-center gap-2 mb-3">
          <h1 class="h3 mb-0 me-auto">{{ d.name }}</h1>
          @if (session.isInstructor()) {
            <button type="button" class="btn btn-outline-secondary btn-sm" (click)="startRename()">Rename</button>
          }
          @if (cards().length > 0) {
            <a class="btn btn-primary btn-sm" [routerLink]="['/decks', d.id, 'study']">Study this deck</a>
          }
        </div>
      }

      @if (session.isInstructor()) {
        <form class="card card-body mb-4" (ngSubmit)="add()">
          <h2 class="h6">Add a card</h2>
          <div class="row g-2">
            <div class="col-md-6">
              <label for="new-front" class="form-label">Front</label>
              <textarea id="new-front" name="front" class="form-control" rows="2" maxlength="1000" required
                        [(ngModel)]="newFront"></textarea>
            </div>
            <div class="col-md-6">
              <label for="new-back" class="form-label">Back</label>
              <textarea id="new-back" name="back" class="form-control" rows="2" maxlength="1000" required
                        [(ngModel)]="newBack"></textarea>
            </div>
          </div>
          <div class="mt-2">
            <button type="submit" class="btn btn-primary" [disabled]="!newFront().trim() || !newBack().trim()">
              Add card
            </button>
          </div>
        </form>
      }

      @if (cards().length === 0) {
        <p class="text-body-secondary">This deck has no cards yet.</p>
      } @else {
        <ul class="list-group">
          @for (card of cards(); track card.id; let i = $index) {
            <li class="list-group-item">
              @if (editingId() === card.id) {
                <form (ngSubmit)="save(card)">
                  <div class="row g-2">
                    <div class="col-md-6">
                      <label [for]="'front-' + card.id" class="form-label">Front</label>
                      <textarea [id]="'front-' + card.id" name="front" class="form-control" rows="2"
                                maxlength="1000" required [(ngModel)]="editFront"></textarea>
                    </div>
                    <div class="col-md-6">
                      <label [for]="'back-' + card.id" class="form-label">Back</label>
                      <textarea [id]="'back-' + card.id" name="back" class="form-control" rows="2"
                                maxlength="1000" required [(ngModel)]="editBack"></textarea>
                    </div>
                  </div>
                  <div class="mt-2 d-flex gap-2">
                    <button type="submit" class="btn btn-primary btn-sm"
                            [disabled]="!editFront().trim() || !editBack().trim()">Save</button>
                    <button type="button" class="btn btn-outline-secondary btn-sm" (click)="editingId.set(null)">
                      Cancel
                    </button>
                  </div>
                </form>
              } @else {
                <div class="d-flex gap-3">
                  <!-- Zero flex-basis: the text columns take the space left after the buttons, so every
                       row's columns line up and long answers wrap instead of pushing the buttons down. -->
                  <div class="row" style="flex: 1 1 0; min-width: 0; overflow-wrap: anywhere">
                    <div class="col-md-6"><span class="visually-hidden">Front: </span>{{ card.front }}</div>
                    <div class="col-md-6 text-body-secondary"><span class="visually-hidden">Back: </span>{{ card.back }}</div>
                  </div>
                  @if (session.isInstructor()) {
                    <div class="d-flex flex-shrink-0 gap-2 align-items-start">
                      <button type="button" class="btn btn-outline-secondary btn-sm"
                              [attr.aria-label]="'Edit card ' + (i + 1)" (click)="startEdit(card)">Edit</button>
                      @if (confirmingId() === card.id) {
                        <button type="button" class="btn btn-danger btn-sm" (click)="remove(card)">Confirm delete</button>
                        <button type="button" class="btn btn-link btn-sm" (click)="confirmingId.set(null)">Cancel</button>
                      } @else {
                        <button type="button" class="btn btn-outline-danger btn-sm"
                                [attr.aria-label]="'Delete card ' + (i + 1)"
                                (click)="confirmingId.set(card.id)">Delete</button>
                      }
                    </div>
                  }
                </div>
              }
            </li>
          }
        </ul>
      }
    }
  `,
})
export class DeckView implements OnInit {
  /** The :id route parameter (bound by withComponentInputBinding). */
  readonly id = input.required<string>();

  protected readonly session = inject(SessionService);
  private readonly api = inject(FlashcardsService);

  protected readonly deck = signal<Deck | null>(null);
  private readonly title = inject(Title);
  /** Keep the tab title in step with the deck name, including after a rename. */
  private readonly syncTitle = effect(() => {
    const name = this.deck()?.name;
    if (name) this.title.setTitle(`${name} · Flashcards`);
  });
  protected readonly cards = signal<Card[]>([]);
  protected readonly loading = signal(true);
  protected readonly error = signal<string | null>(null);

  protected readonly renaming = signal(false);
  protected readonly deckName = signal('');
  protected readonly newFront = signal('');
  protected readonly newBack = signal('');
  protected readonly editingId = signal<number | null>(null);
  protected readonly editFront = signal('');
  protected readonly editBack = signal('');
  protected readonly confirmingId = signal<number | null>(null);

  private get deckId(): number {
    return Number(this.id());
  }

  ngOnInit(): void {
    forkJoin([this.api.getDeck(this.deckId), this.api.listCards(this.deckId)]).subscribe({
      next: ([deck, cards]) => {
        this.deck.set(deck);
        this.cards.set(cards);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(errorMessage(err));
        this.loading.set(false);
      },
    });
  }

  protected startRename(): void {
    this.deckName.set(this.deck()?.name ?? '');
    this.renaming.set(true);
  }

  protected rename(): void {
    this.error.set(null);
    this.api.updateDeck(this.deckId, this.deckName().trim()).subscribe({
      next: (deck) => {
        this.deck.set(deck);
        this.renaming.set(false);
      },
      error: (err) => this.error.set(errorMessage(err)),
    });
  }

  protected add(): void {
    this.error.set(null);
    this.api.createCard(this.deckId, { front: this.newFront().trim(), back: this.newBack().trim() }).subscribe({
      next: (card) => {
        this.cards.update((cs) => [...cs, card]);
        this.newFront.set('');
        this.newBack.set('');
        document.getElementById('new-front')?.focus();
      },
      error: (err) => this.error.set(errorMessage(err)),
    });
  }

  protected startEdit(card: Card): void {
    this.editFront.set(card.front);
    this.editBack.set(card.back);
    this.confirmingId.set(null);
    this.editingId.set(card.id);
  }

  protected save(card: Card): void {
    this.error.set(null);
    this.api.updateCard(card.id, { front: this.editFront().trim(), back: this.editBack().trim() }).subscribe({
      next: (updated) => {
        this.cards.update((cs) => cs.map((c) => (c.id === updated.id ? updated : c)));
        this.editingId.set(null);
      },
      error: (err) => this.error.set(errorMessage(err)),
    });
  }

  protected remove(card: Card): void {
    this.error.set(null);
    this.api.deleteCard(card.id).subscribe({
      next: () => {
        this.cards.update((cs) => cs.filter((c) => c.id !== card.id));
        this.confirmingId.set(null);
      },
      error: (err) => this.error.set(errorMessage(err)),
    });
  }
}
