import { Component, OnInit, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { Deck, FlashcardsService, errorMessage } from './flashcards.service';
import { SessionService } from './session';

@Component({
  selector: 'app-deck-list',
  imports: [FormsModule, RouterLink],
  template: `
    <h1 class="h3 mb-3">Decks</h1>

    @if (error()) {
      <div class="alert alert-danger" role="alert">{{ error() }}</div>
    }

    @if (session.isInstructor()) {
      <form class="row g-2 align-items-end mb-4" (ngSubmit)="create()">
        <div class="col-sm-8 col-md-6">
          <label for="new-deck" class="form-label">New deck name</label>
          <input id="new-deck" name="name" class="form-control" maxlength="100" required
                 [(ngModel)]="newName" />
        </div>
        <div class="col-auto">
          <button type="submit" class="btn btn-primary" [disabled]="saving() || !newName().trim()">
            Add deck
          </button>
        </div>
      </form>
    }

    @if (loading()) {
      <div class="d-flex align-items-center gap-2" role="status">
        <div class="spinner-border spinner-border-sm" aria-hidden="true"></div>
        <span>Loading decks…</span>
      </div>
    } @else if (decks().length === 0) {
      <p class="text-body-secondary">
        No decks yet.
        @if (session.isInstructor()) { Add one above. } @else { Your instructor hasn't added any. }
      </p>
    } @else {
      <div class="row row-cols-1 row-cols-md-2 row-cols-lg-3 g-3">
        @for (deck of decks(); track deck.id) {
          <div class="col">
            <div class="card h-100">
              <div class="card-body">
                <h2 class="h5 card-title">{{ deck.name }}</h2>
              </div>
              <div class="card-footer d-flex flex-wrap gap-2">
                <a class="btn btn-primary btn-sm" [routerLink]="['/decks', deck.id, 'study']">Study</a>
                <a class="btn btn-outline-secondary btn-sm" [routerLink]="['/decks', deck.id]">
                  {{ session.isInstructor() ? 'Edit cards' : 'View cards' }}
                </a>
                @if (session.isInstructor()) {
                  @if (confirmingId() === deck.id) {
                    <button type="button" class="btn btn-danger btn-sm" (click)="remove(deck)">
                      Confirm delete
                    </button>
                    <button type="button" class="btn btn-link btn-sm" (click)="confirmingId.set(null)">
                      Cancel
                    </button>
                  } @else {
                    <button type="button" class="btn btn-outline-danger btn-sm ms-auto"
                            [attr.aria-label]="'Delete deck ' + deck.name"
                            (click)="confirmingId.set(deck.id)">
                      Delete
                    </button>
                  }
                }
              </div>
            </div>
          </div>
        }
      </div>
    }
  `,
})
export class DeckList implements OnInit {
  protected readonly session = inject(SessionService);
  private readonly api = inject(FlashcardsService);

  protected readonly decks = signal<Deck[]>([]);
  protected readonly loading = signal(true);
  protected readonly saving = signal(false);
  protected readonly error = signal<string | null>(null);
  protected readonly newName = signal('');
  protected readonly confirmingId = signal<number | null>(null);

  ngOnInit(): void {
    this.api.listDecks().subscribe({
      next: (decks) => {
        this.decks.set(decks);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(errorMessage(err));
        this.loading.set(false);
      },
    });
  }

  protected create(): void {
    const name = this.newName().trim();
    if (!name) return;
    this.saving.set(true);
    this.error.set(null);
    this.api.createDeck(name).subscribe({
      next: (deck) => {
        this.decks.update((ds) => [...ds, deck]);
        this.newName.set('');
        this.saving.set(false);
      },
      error: (err) => {
        this.error.set(errorMessage(err));
        this.saving.set(false);
      },
    });
  }

  protected remove(deck: Deck): void {
    this.error.set(null);
    this.api.deleteDeck(deck.id).subscribe({
      next: () => {
        this.decks.update((ds) => ds.filter((d) => d.id !== deck.id));
        this.confirmingId.set(null);
      },
      error: (err) => this.error.set(errorMessage(err)),
    });
  }
}
