import { Component, HostListener, OnInit, computed, inject, input, signal } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { RouterLink } from '@angular/router';
import { forkJoin } from 'rxjs';
import { Card, Deck, FlashcardsService, errorMessage } from './flashcards.service';

/** Study mode: one card at a time, front first; Flip shows the back. */
@Component({
  selector: 'app-study',
  imports: [RouterLink],
  template: `
    <nav aria-label="breadcrumb">
      <ol class="breadcrumb">
        <li class="breadcrumb-item"><a routerLink="/">Decks</a></li>
        <li class="breadcrumb-item"><a [routerLink]="['/decks', id()]">{{ deck()?.name ?? 'Deck' }}</a></li>
        <li class="breadcrumb-item active" aria-current="page">Study</li>
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
    } @else if (current(); as card) {
      <p class="text-body-secondary text-center mb-2">Card {{ index() + 1 }} of {{ cards().length }}</p>

      <div class="d-flex gap-2 mb-3">
        <!-- aria-disabled rather than disabled so focus isn't dropped at the first/last card. -->
        <button type="button" class="btn btn-outline-secondary study-nav" (click)="move(-1)"
                aria-label="Previous card" [class.disabled]="isFirst()" [attr.aria-disabled]="isFirst()">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"
               stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">
            <path d="M15 6l-6 6 6 6" />
          </svg>
        </button>

        <!-- Re-created per card so the old one can slide out while the new one slides in. -->
        <div class="study-stage" [class.dir-prev]="direction() < 0">
          @for (c of [card]; track c.id) {
            <div class="study-card" (click)="cardClick()" [class.flipped]="flipped()"
                 [animate.enter]="moved() ? 'slide-in' : ''" animate.leave="slide-out">
              <div class="study-inner">
                <div class="card study-face study-front" [attr.aria-hidden]="flipped()" [inert]="flipped()">
                  <div class="card-body d-flex flex-column justify-content-center text-center">
                    <p class="fs-4 mb-0 study-text"><span class="visually-hidden">Front: </span>{{ c.front }}</p>
                  </div>
                </div>
                <div class="card study-face study-back" [attr.aria-hidden]="!flipped()" [inert]="!flipped()">
                  <div class="card-body d-flex flex-column justify-content-center text-center">
                    <p class="fs-4 mb-0 study-text"><span class="visually-hidden">Back: </span>{{ c.back }}</p>
                  </div>
                </div>
              </div>
            </div>
          }
        </div>

        <button type="button" class="btn btn-outline-secondary study-nav" (click)="move(1)"
                aria-label="Next card" [class.disabled]="isLast()" [attr.aria-disabled]="isLast()">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"
               stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">
            <path d="M9 6l6 6-6 6" />
          </svg>
        </button>
      </div>

      <div class="d-flex justify-content-center">
        <button type="button" class="btn btn-primary px-4" (click)="flip()">
          {{ flipped() ? 'Show front' : 'Show back' }}
        </button>
      </div>
      <p class="small text-body-secondary text-center mt-3 mb-0">
        Keyboard: <kbd>Space</kbd> flips the card, <kbd>←</kbd> <kbd>→</kbd> move between cards.
      </p>
      <p class="visually-hidden" aria-live="polite">{{ announcement() }}</p>
    } @else if (!error()) {
      <p class="text-body-secondary">This deck has no cards to study yet.</p>
    }
  `,
  styles: `
    .study-stage {
      flex: 1 1 auto;
      min-width: 0;
      display: grid;
      /* Clip horizontally only, so the 3D flip isn't cut off top and bottom. */
      overflow-x: clip;
    }
    .study-card { grid-area: 1 / 1; perspective: 1200px; cursor: pointer; }
    .slide-in { animation: slide-in-from-right 0.3s ease-out; }
    .slide-out { animation: slide-out-to-left 0.3s ease-in forwards; pointer-events: none; }
    .dir-prev .slide-in { animation-name: slide-in-from-left; }
    .dir-prev .slide-out { animation-name: slide-out-to-right; }
    @keyframes slide-in-from-right { from { transform: translateX(100%); } }
    @keyframes slide-in-from-left { from { transform: translateX(-100%); } }
    @keyframes slide-out-to-left { to { transform: translateX(-100%); } }
    @keyframes slide-out-to-right { to { transform: translateX(100%); } }
    .study-nav {
      flex: 0 0 3rem;
      padding: 0;
      display: flex;
      align-items: center;
      justify-content: center;
    }
    /* Keep aria-disabled buttons focusable; Bootstrap's .disabled would otherwise block pointer events only. */
    .btn.disabled { pointer-events: auto; cursor: not-allowed; }
    .study-inner {
      display: grid;
      transform-style: preserve-3d;
      transition: transform 0.5s ease;
    }
    .study-card.flipped .study-inner { transform: rotateY(180deg); }
    .study-face {
      grid-area: 1 / 1;
      min-height: 14rem;
      backface-visibility: hidden;
      -webkit-backface-visibility: hidden;
    }
    .study-front { transform: rotateY(0deg); background-color: #0b2545; border-color: #0b2545; color: #fff; }
    /* Explicit text colour so the white back stays readable under a dark theme. */
    .study-back { background-color: #fff; color: #212529; border-color: #dee2e6; transform: rotateY(180deg); }
    .study-text { white-space: pre-wrap; overflow-wrap: anywhere; }
    @media (prefers-reduced-motion: reduce) {
      .study-inner { transition: none; }
      .slide-in, .slide-out { animation: none; }
    }
  `,
})
export class Study implements OnInit {
  readonly id = input.required<string>();

  private readonly api = inject(FlashcardsService);
  private readonly title = inject(Title);

  protected readonly deck = signal<Deck | null>(null);
  protected readonly cards = signal<Card[]>([]);
  protected readonly index = signal(0);
  protected readonly flipped = signal(false);
  /** Direction of the last card change: 1 = next, -1 = previous. Picks the slide direction. */
  protected readonly direction = signal(1);
  /** False until the first card change, so the first card doesn't slide in on page load. */
  protected readonly moved = signal(false);
  protected readonly loading = signal(true);
  protected readonly error = signal<string | null>(null);
  protected readonly current = computed(() => this.cards()[this.index()] ?? null);
  protected readonly isFirst = computed(() => this.index() === 0);
  protected readonly isLast = computed(() => this.index() >= this.cards().length - 1);
  /** Text for the screen-reader live region, updated on each flip or card change. */
  protected readonly announcement = signal('');

  ngOnInit(): void {
    const id = Number(this.id());
    forkJoin([this.api.getDeck(id), this.api.listCards(id)]).subscribe({
      next: ([deck, cards]) => {
        this.deck.set(deck);
        this.title.setTitle(`Study: ${deck.name} · Flashcards`);
        this.cards.set(cards);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(errorMessage(err));
        this.loading.set(false);
      },
    });
  }

  protected move(step: number): void {
    const next = this.index() + step;
    if (next >= 0 && next < this.cards().length) {
      this.direction.set(step);
      this.moved.set(true);
      this.index.set(next);
      this.flipped.set(false);
      const card = this.cards()[next];
      this.announcement.set(`Card ${next + 1} of ${this.cards().length}. Front: ${card.front}`);
    }
  }

  protected flip(): void {
    const card = this.current();
    if (!card) return;
    this.flipped.set(!this.flipped());
    this.announcement.set(this.flipped() ? `Back: ${card.back}` : `Front: ${card.front}`);
  }

  /** Clicking the card flips it, unless the click ended a text selection. */
  protected cardClick(): void {
    if (window.getSelection()?.toString()) return;
    this.flip();
  }

  @HostListener('document:keydown', ['$event'])
  protected onKeydown(e: KeyboardEvent): void {
    if (e.defaultPrevented || e.ctrlKey || e.metaKey || e.altKey || !this.current()) return;
    const target = e.target as HTMLElement | null;
    if (target?.closest('input, textarea, select, [contenteditable]')) return;

    switch (e.key) {
      case 'ArrowLeft':
        e.preventDefault();
        this.move(-1);
        break;
      case 'ArrowRight':
        e.preventDefault();
        this.move(1);
        break;
      case ' ':
        // Buttons and links already act on Space; don't double-handle it.
        if (target?.closest('button, a, [role="button"]')) return;
        e.preventDefault();
        this.flip();
        break;
    }
  }
}
