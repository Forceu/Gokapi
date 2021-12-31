/*
 * Copyright (C) 2013 Panopta, Andrew Moffat
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */
(function ($) {
    $.fn.wizard = function(args) {
        return new Wizard(this, args);
    };

    $.fn.wizard.logging = false;
    
    var WizardCard = function(wizard, card, index, prev, next) {
        this.wizard 	= wizard;
        this.index 		= index;
        this.prev 		= prev;
        this.next 		= next;
        this.el 		= card;
        this.title 		= card.find("h3").first().text();
        this.name 		= card.data("cardname") || this.title;

        this.nav 		= this._createNavElement(this.title, index);

        this._disabled 	= false;
        this._loaded 	= false;
        this._events =	 {};
    };
    
    WizardCard.prototype = {
        select: function() {
            this.log("selecting");
            if (!this.isSelected()) {
                this.nav.addClass("active");
                this.el.show();

                if (!this._loaded) {
                    this.trigger("loaded");
                    this.reload();
                }

                this.trigger("selected");
            }


            /*
             * this is ugly, but we're handling the changing of the wizard's
             * buttons here, in the WizardCard select.  so when a card is
             * selected, we're figuring out if we're the first card or the
             * last card and changing the wizard's buttons via the guts of
             * the wizard
             *
             * ideally this logic should be encapsulated by some wizard methods
             * that we can call from here, instead of messing with the guts
             */
            var w = this.wizard;

            // The back button is only disabled on this first card...
            w.backButton.toggleClass("disabled", this.index == 0);

            if (this.index >= w._cards.length-1) {
                this.log("on last card, changing next button to submit");

                w.changeNextButton(w.args.buttons.submitText, "btn-success");
                w._readyToSubmit = true;
                w.trigger("readySubmit");
            }
            else {
                w._readyToSubmit = false;
                w.changeNextButton(w.args.buttons.nextText, "btn-primary");
            }

            return this;
        },

        _createNavElement: function(name, i) {
            var li = $('<li class="wizard-nav-item"></li>');
            var a = $('<a class="wizard-nav-link"></a>');
            a.data("navindex", i);
            li.append(a);
            a.append('<span class="glyphicon glyphicon-chevron-right"></span> ');
            a.append(name);
            return li;
        },

        markVisited: function() {
            this.log("marking as visited");
            this.nav.addClass("already-visited");
            this.trigger("markVisited");
            return this;
        },

        unmarkVisited: function() {
            this.log("unmarking as visited");
            this.nav.removeClass("already-visited");
            this.trigger("unmarkVisited");
            return this;
        },

        deselect: function() {
            this.nav.removeClass("active");
            this.el.hide();
            this.trigger("deselect");
            return this;
        },

      enable: function(noActivate) {
            this.log("enabling");
            this.nav.removeClass('hide');
            if (noActivate !== undefined) {
                this.nav.addClass("active");
            }
            this._disabled = false;
            this.trigger("enabled");
            return this;
        },

        disable: function(hideCard) {
            this.log("disabling");
            this._disabled = true;
            this.nav.removeClass("active already-visited");
            if (hideCard) {
                this.el.hide();
                this.nav.addClass('hide');
            }
            this.trigger("disabled");
            return this;
        },

        isDisabled: function() {
            return this._disabled;
        },

        alreadyVisited: function() {
            return this.nav.hasClass("already-visited");
        },

        isSelected: function() {
            return this.nav.hasClass("active");
        },

        reload: function() {
            this._loaded = true;
            this.trigger("reload");
            return this;
        },

        on: function() {
            return this.wizard.on.apply(this, arguments);
        },

        trigger: function() {
            this.callListener("on"+arguments[0]);
            return this.wizard.trigger.apply(this, arguments);
        },

        /*
         * displays an alert box on the current card
         */
        toggleAlert: function(msg, toggle) {
            this.log("toggling alert to: " + toggle);

            toggle = typeof(toggle) == "undefined" ? true : toggle;

            if (toggle) {this.trigger("showAlert");}
            else {this.trigger("hideAlert");}

            var div;
            var alert = this.el.children("h3").first().next("div.alert");

            if (alert.length == 0) {
                /*
                 * we're hiding anyways, so no need to create anything.
                 * we'll do that if we ever are actually showing the alert
                 */
                if (!toggle) {return this;}

                this.log("couldn't find existing alert div, creating one");
                div = $("<div />");
                div.addClass("alert");
                div.addClass("hide");
                div.insertAfter(this.el.find("h3").first());
            }
            else {
                this.log("found existing alert div");
                div = alert.first();
            }

            if (toggle) {
                if (msg != null) {
                    this.log("setting alert msg to", msg);
                    div.html(msg);
                }
                div.show();
            }
            else {
                div.hide();
            }
            return this;
        },

        /*
         * this looks for event handlers embedded into the html of the
         * wizard card itself, in the form of a data- attribute
         */
        callListener: function(name) {
            // a bug(?) in jquery..can't access data-<name> if name is camelCase
            name = name.toLowerCase();

            this.log("looking for listener " + name);
            var listener = window[this.el.data(name)];
            if (listener) {
                this.log("calling listener " + name);
                var wizard = this.wizard;

                try {
                    var vret = listener(this);
                }
                catch (e) {
                    this.log("exception calling listener " + name + ": ", e);
                }
            }
            else {
                this.log("didn't find listener " + name);
            }
        },

        problem: function(toggle) {
            this.nav.find("a").toggleClass("wizard-step-error", toggle);
        },

        validate: function() {
            var failures = false;
            var self = this;

            /*
             * run all the validators embedded on the inputs themselves
             */
            this.el.find("[data-validate]").each(function(i, el) {
                self.log("validating individiual inputs");
                el = $(el);

                var v = el.data("validate");
                if (!v) {return;}

                var ret = {
                    status: true,
                    title: "Error",
                    msg: ""
                };

                var vret = window[v](el);
                $.extend(ret, vret);

                // Add-On
                // This allows the use of a INPUT+BTN used as one according to boostrap layout
                // for the wizard it is required to add an id with btn-(ID of Input)
                // this will make sure the popover is drawn on the correct element
                if ( $('#btn-' + el.attr('id')).length === 1 ) {
                    el = $('#btn-' + el.attr('id'));
                }

                if (!ret.status) {
                    failures = true;
                    
                    // Updated to show error on correct form-group
                    el.parents("div.form-group").toggleClass("has-error", true);
                    
                    // This allows the use of a INPUT+BTN used as one according to boostrap layout
                    // for the wizard it is required to add an id with btn-(ID of Input)
                    // this will make sure the popover is drawn on the correct element
                    if ( $('#btn-' + el.attr('id')).length === 1 ) {
                        el = $('#btn-' + el.attr('id'));
                    }
                    
                    self.wizard.errorPopover(el, ret.msg);
                } else {
                    el.parents("div.form-group").toggleClass("has-error", false);
                    
                    // This allows the use of a INPUT+BTN used as one according to boostrap layout
                    // for the wizard it is required to add an id with btn-(ID of Input)
                    // this will make sure the popover is drawn on the correct element
                    if ( $('#btn-' + el.attr('id')).length === 1 ) {
                        el = $('#btn-' + el.attr('id'));
                    }
                    
                    try {
                        el.popover("destroy");
                    }
                    /*
                     * older versions of bootstrap don't have a destroy call
                     * for popovers
                     */
                    catch (e) {
                        el.popover("hide");
                    }
                }
            });
            this.log("after validating inputs, failures is", failures);

            /*
             * run the validator embedded in the card
             */
            var cardValidator = window[this.el.data("validate")];
            if (cardValidator) {
                this.log("running html-embedded card validator");
                var cardValidated = cardValidator(this);
                if (typeof(cardValidated) == "undefined" || cardValidated == null) {
                    cardValidated = true;
                }
                if (!cardValidated) failures = true;
                this.log("after running html-embedded card validator, failures is", failures);
            }

            /*
             * run the validate listener
             */
            this.log("running listener validator");
            var listenerValidated = this.trigger("validate");
            if (typeof(listenerValidated) == "undefined" || listenerValidated == null) {
                listenerValidated = true;
            }
            if (!listenerValidated) failures = true;
            this.log("after running listener validator, failures is", failures);

            var validated = !failures;
            if (validated) {
                this.log("validated, calling listeners");
                this.trigger("validated");
            }
            else {
                this.log("invalid");
                this.trigger("invalid");
            }
            return validated;
        },
        
        log: function() {
            if (!window.console || !$.fn.wizard.logging) {return;}
            var prepend = "card '"+this.name+"': ";
            var args = [prepend];
            args.push.apply(args, arguments);

            console.log.apply(console, args);
        },

        isActive: function() {
            return this.nav.hasClass("active");
        }
    };
    
    Wizard = function(markup, args) {
        
        /* TEMPLATE */
        this.wizard_template = [
            '<div  class="modal fade wizard">',
                '<div class="modal-dialog wizard-dialog">',
                    '<div class="modal-content wizard-content">',
                        '<div class="modal-header wizard-header">',
                            '<button type="button" class="close wizard-close" aria-hidden="true">&times;</button>',
                            '<h3 class="modal-title wizard-title"></h3>',
                            '<span class="wizard-subtitle"></span>',
                        '</div>',
                        '<div class="modal-body wizard-body">',
                            '<div class="pull-left wizard-steps">',
                                '<div class="wizard-nav-container">',
                                    '<ul class="nav wizard-nav-list">',
                                    '</ul>',
                                '</div>',
                                '<div class="wizard-progress-container">',
                                    '<div class="progress progress-striped">',
                                        '<div class="progress-bar" style="width: 0%;"></div>',
                                    '</div>',
                                '</div>',
                            '</div>',
                            '<form>',
                                '<div class="wizard-cards">',
                                    '<div class="wizard-card-container">',
                                    '</div>',
                                    '<div class="wizard-footer">',
                                        '<div class="wizard-buttons-container">',
                                            '<button class="btn wizard-cancel wizard-close" type="button">Cancel</button>',
                                            '<div class="btn-group-single pull-right">',
                                                '<button class="btn wizard-back" type="button">Back</button>',
                                                '<button class="btn btn-primary wizard-next" type="button">Next</button>',
                                            '</div>',
                                        '</div>',
                                    '</div>',
                                '</div>',
                            '</form>',
                        '</div>',
                    
                    '</div>',
                '</div>',
            '</div>'
        ];
        
        this.args = {
            keyboard: true,
            backdrop: true,
            show: false,
            submitUrl: "",
            showCancel: false,
            showClose: true,
            progressBarCurrent: false,
            increaseHeight: 0,
            contentHeight: 300,
            contentWidth: 580,
            buttons: {
                cancelText: "Cancel",
                nextText: "Next",
                backText: "Back",
                submitText: "Submit",
                submittingText: "Submitting...",
            },
            formClass: "form-horizontal"
        };
        
        $.extend(this.args, args || {});
        
        this._create(markup);
    };
    
    Wizard.prototype = {
        log: function() {
            if (!window.console || !$.fn.wizard.logging) {return;}
            var prepend = "wizard: ";
            var args = [prepend];
            args.push.apply(args, arguments);
            console.log.apply(console, args);
        },
        
        _create: function(markup) {
            this.markup = $(markup);
            this.title					= 	this.markup.data('title');
            this.submitCards 			= 	this.markup.find(".wizard-error,.wizard-failure,.wizard-success,.wizard-loading");
            this.el						=	$(this.wizard_template.join('\n'));
            $('body').append(this.el);
            
            this.modal 					= 	this.el.modal({
                keyboard: this.args.keyboard,
                show: this.args.show,
                backdrop: this.args.backdrop
            });
            
            this.dimensions				=	{
                                                contentHeight: this.args.contentHeight,
                                                contentWidth: this.args.contentWidth
                                            };
            this.dialog 				=	this.modal.find('.wizard-dialog');
            this.content 				= 	this.modal.find('.wizard-content');
            this.header 				= 	this.modal.find('.wizard-header');
            this.body 					= 	this.modal.find('.wizard-body');
            this.wizardSteps			= 	this.modal.find('.wizard-steps');
            this.wizardCards			=	this.modal.find('.wizard-cards');
            this.wizardCardContainer	=	this.modal.find('.wizard-card-container');
            this.wizardCardContainer
                .append(this.markup.find('.wizard-card'))
                .append(this.submitCards);
            this.navContainer 			= 	this.modal.find('.wizard-nav-container');
            this.navList				= 	this.modal.find('.wizard-nav-list');
            this.progressContainer		= 	this.modal.find('.wizard-progress-container');
            this.progress				= 	this.progressContainer.find('.progress-bar');
            this.closeButton 			= 	this.modal.find('button.wizard-close.close');
            this.cardsContainer			=	this.modal.find('wizard-cards-container');
            this.form					=	this.modal.find('form');
            this.footer 				= 	this.modal.find(".wizard-footer");
            this.cancelButton 			= 	this.footer.find(".wizard-cancel");
            this.backButton 			= 	this.footer.find(".wizard-back");
            this.nextButton 			= 	this.footer.find(".wizard-next");
            
            this._cards 				= 	[];
            this.cards 					= 	{};
            this._readyToSubmit 		= 	false;
            this.percentComplete 		=	0;
            this._submitting 			= 	false;
            this._events 				= 	{};
            this._firstShow 			= 	true;
            
            this._createCards();

            this.nextButton.click(this, this._handleNextClick);
            this.backButton.click(this, this._handleBackClick);

            this.cancelButton.text(this.args.buttons.cancelText);
            this.backButton.text(this.args.buttons.backText);
            this.nextButton.text(this.args.buttons.nextText);
            
            // Apply Form Class(es)
            this.form.addClass(this.args.formClass);
            
            // Register Array Holder for popovers
            this.popovers				= [];

            var self = this;
            var _close = function() {
                self.reset();
                self.close();
                self.trigger("closed");
            };

            // Register Close Button
            this.closeButton.click(_close);
            this.cancelButton.click(_close);
            
            this.wizardSteps.on("click", "li.already-visited a.wizard-nav-link", this,
            function(event) {
                var index = parseInt($(event.target).data("navindex"));
                event.data.setCard(index);
            });
            
            if ( this.title.length != 0 ) {
                this.setTitle(this.title);
            }
            
            this.on("submit", this._defaultSubmit);
            
            // Set Modal Dimensions
            this.autoDimensions();
        },
        
        autoDimensions: function() {
            // DO NOT REMOVE DISPLAY ; Temporary display is required for calculation
            this.modal.css('display', 'block');
            
            this.dimensions.header = this.header.outerHeight(true);
            
            // Navigation Pane is dyanmic build on card content
            // Navigation Pane === BASE Inner Content Height
            this.dimensions.navigation = this.wizardSteps.outerHeight(true);
            if ( this.dimensions.navigation < this.dimensions.contentHeight ) {
                this.dimensions.navigation = this.dimensions.contentHeight;
                this.navContainer.height( (this.dimensions.contentHeight-30) - this.progressContainer.outerHeight(true));
            }
            
            // Dimension Alias ( Body Height === (Navigation Height) )
            this.dimensions.body = this.dimensions.navigation;
            
            // Apply OuterHeight of navigation to it's parent wizardSteps
            this.wizardSteps.height(this.dimensions.body);
            
            // Modal Height === (Header + Content)
            this.dimensions.modal = (this.dimensions.header + this.dimensions.navigation);
            this.content.height(this.dimensions.modal + 'px');
            this.dialog.width(this.dimensions.contentWidth);
            
            this.body.height(this.dimensions.body + 'px');
            this.wizardCards.height(this.dimensions.body + 'px');
            
            // Footer Height
            this.dimensions.footer = this.footer.outerHeight(true);
            
            // Card Container === (Body - Footer)
            this.dimensions.cardContainer = (this.dimensions.body - this.dimensions.footer);
            this.wizardCardContainer.height(this.dimensions.cardContainer);
            
            // Reposition
            this.dimensions.offset = ($(window).height() - this.dialog.height()) / 2;			
            this.dialog.css({
                'margin-top': this.dimensions.offset + 'px',
                'padding-top': 0
            });
            
            // DO NOT REMOVE NEXT LINE
            this.modal.css('display', '');
        },
        
        setTitle: function(title) {
            this.log("setting title to", title);
            this.modal.find(".wizard-title").first().text(title);
            return this;
        },

        setSubtitle: function(title) {
            this.log("setting subtitle to", title);
            this.modal.find(".wizard-subtitle").first().text(title);
            return this;
        },
        
        errorPopover: function(el, msg, allowHtml) {
            this.log("launching popover on", el);
            allowHtml = typeof allowHtml !== "undefined" ? allowHtml : false;
            var popover = el.popover('destroy').popover({
                content: msg,
                trigger: "manual",
                html: allowHtml,
                container: el.parents('.form-group')
            }).addClass("error-popover").popover("show").next(".popover");

            el.parents('.form-group').find('.popover').addClass("error-popover");
            
            this.popovers.push(el);
            
            return popover;
        },
        
        destroyPopover: function(pop) {
            pop = $(pop);
            
            /*
             * this is the element that the popover was created for
             */
            try {
                pop.popover("destroy");
            }
            /*
             * older versions of bootstrap don't have a destroy call
             * for popovers
             */
            catch (e) {
                pop.popover("hide");
            }
        },
        
        hidePopovers: function(el) {
            this.log("hiding all popovers");
            var self = this;

            $.each(this.popovers, function(i, p) {
                self.destroyPopover(p);
            });
            
            this.modal.find('.has-error').removeClass('has-error');
            this.popovers = [];
        },

        eachCard: function(fn) {
            $.each(this._cards, fn);
            return this;
        },

        getActiveCard: function() {
            this.log("getting active card");
            var currentCard = null;

            $.each(this._cards, function(i, card) {
                if (card.isActive()) {
                    currentCard = card;
                    return false;
                }
            });
            if (currentCard) {this.log("found active card", currentCard);}
            else {this.log("couldn't find an active card");}
            return currentCard;
        },

        changeNextButton: function(text, cls) {
            this.log("changing next button, text: " + text, "class: " + cls);
            if (typeof(cls) != "undefined") {
                this.nextButton.removeClass("btn-success btn-primary");
            }

            if (cls) {
                this.nextButton.addClass(cls);
            }
            this.nextButton.text(text);
            return this;
        },

        hide: function() {
            this.log("hiding");
            this.modal.modal("hide");
            return this;
        },

        close: function() {
            this.log("closing");
            this.modal.modal("hide");
            return this;
        },


        show: function(modalOptions) {
            this.log("showing");
            if (this._firstShow) {
                this.setCard(0);
                this._firstShow = false;
            }
            if (this.args.showCancel) { 
                this.cancelButton.show(); 
            } else {
                this.cancelButton.hide(); 
            }
            if (this.args.showClose) { this.closeButton.show(); }  else {
                this.closeButton.hide();
            }
            this.modal.modal('show');
            
            return this;
        },

        on: function(name, fn) {
            this.log("adding listener to event " + name);
            this._events[name] = fn;
            return this;
        },

        trigger: function() {
            var name = arguments[0];
            var args = Array.prototype.slice.call(arguments);
            args.shift();
            args.unshift(this);

            this.log("firing event " + name);
            var handler = this._events[name];
            if (handler === undefined && this.wizard !== undefined) {
                handler = this.wizard._events[name];
            }
            var ret = null;

            if (typeof(handler) == "function") {
                this.log("found event handler, calling " + name);
                try {
                    ret = handler.apply(this, args);
                }
                catch (e) {
                    this.log("event handler " + name + " had an exception");
                }
            }
            else {
                this.log("couldn't find an event handler for " + name);
            }
            return ret;
        },


        reset: function() {
            this.log("resetting");
            
            this.updateProgressBar(0);
            this.hideSubmitCards();

            this.setCard(0);
            this.lockCards();

            this.enableNextButton();
            this.showButtons();
            
            this.hidePopovers();

            this.trigger("reset");
            return this;
        },
        
        /*
         * this handles switching to the next card or previous card, taking
         * care to skip over disabled cards
         */
        _abstractIncrementStep: function(direction, getNext) {
            var current = this.getActiveCard();
            var next;

            if (current) {
                /*
                 * loop until we find a card that isn't disabled
                 */
                this.log("searching for valid next card");
                while (true) {
                    next = getNext(current);
                    if (next) {
                        this.log("looking at card", next.index);
                        if (next.isDisabled()) {
                            this.log("card " + next.index + " is disabled/locked, continuing");
                            current = next;
                            continue;
                        }
                        else {
                            return this.setCard(current.index+direction);
                        }
                    }
                    else {
                        this.log("next card is not defined, breaking");
                        break;
                    }
                }
            }
            else {
                this.log("current card is undefined");
            }
        },


        incrementCard: function() {
            this.log("incrementing card");
            var card = this._abstractIncrementStep(1, function(current){return current.next;});
            this.trigger("incrementCard");
            return card;
        },

        decrementCard: function() {
            this.log("decrementing card");
            var card = this._abstractIncrementStep(-1, function(current){return current.prev;});
            this.trigger("decrementCard");
            return card;
        },

        setCard: function(i) {
            this.log("setting card to " + i);
            this.hideSubmitCards();
            var currentCard = this.getActiveCard();

            if (this._submitting) {
                this.log("we're submitting the wizard already, can't change cards");
                return currentCard;
            }

            var newCard = this._cards[i];
            if (newCard) {
                if (newCard.isDisabled()) {
                    this.log("new card is currently disabled, returning");
                    return currentCard;
                }

                if (currentCard) {

                    /*
                     * here, we're only validating if we're going forward,
                     * not if we're going backwards in a step
                     */
                    if (i > currentCard.index) {
                        var cardToValidate = currentCard;
                        var ok = false;

                        /*
                         * we need to loop over every card between our current
                         * card and the card that we clicked, and re-validate
                         * them.  if there's an error, we need to select the
                         * first card to have an error
                         */
                        while (cardToValidate.index != newCard.index) {
                            /*
                             * unless we're validating the card that we're
                             * leaving, we need to select the card, so that
                             * any validators that trigger errorPopovers can
                             * display correctly
                             */
               	      if (cardToValidate.index != currentCard.index) {
                                    cardToValidate.prev.deselect();
                                    cardToValidate.prev.markVisited();
                                    cardToValidate.select();
                                }
                                if (!cardToValidate.isDisabled()) {
                                    ok = cardToValidate.validate();
                                }
                                else {
                                    ok = true;
                                }

                                if (!ok) {
                                    return cardToValidate;
                                }
                            cardToValidate = cardToValidate.next;
                        }

                        cardToValidate.prev.deselect();
                        cardToValidate.prev.markVisited();
                    }

                    currentCard.deselect();
                    currentCard.markVisited();
                }

                newCard.select();

                if (this.args.progressBarCurrent) {
                    this.percentComplete = i * 100.0 / this._cards.length;
                    this.updateProgressBar(this.percentComplete);					
                }
                else {
                    var lastPercent = this.percentComplete;
                    this.percentComplete = i * 100.0 / this._cards.length;
                    this.percentComplete = Math.max(lastPercent, this.percentComplete);
                    this.updateProgressBar(this.percentComplete);
                }

                return newCard;
            }
            else {
                this.log("couldn't find card " + i);
            }
        },

        updateProgressBar: function(percent) {
            this.log("updating progress to " + percent + "%");
            this.progress.css({width: percent + "%"});
            this.percentComplete = percent;

            this.trigger("progressBar", percent);

            if (percent == 100) {
                this.log("progress is 100, animating progress bar");
                this.progressContainer.find('.progress').addClass("active");
            }
            else if (percent == 0) {
                this.log("progress is 0, disabling animation");
                this.progressContainer.find('.progress').removeClass("active");
            }
        },

        getNextCard: function() {
            var currentCard = this.getActiveCard();
            if (currentCard) return currentCard.next;
        },

        lockCards: function() {
            this.log("locking nav cards");
            this.eachCard(function(i,card){card.unmarkVisited();});
            return this;
        },

        disableCards: function() {
            this.log("disabling all nav cards");
            this.eachCard(function(i,card){card.disable();});
            return this;
        },

        enableCards: function() {
            this.log("enabling all nav cards");
            this.eachCard(function(i,card){card.enable();});
            return this;
        },

        hideCards: function() {
            this.log("hiding cards");
            this.eachCard(function(i,card){card.deselect();});
            this.hideSubmitCards();
            return this;
        },

        hideButtons: function() {
            this.log("hiding buttons");
            this.cancelButton.hide();
            this.closeButton.hide();
            this.nextButton.hide();
            this.backButton.hide();
            return this;
        },

        showButtons: function() {
            this.log("showing buttons");
            if (this.args.showCancel) { 
                this.cancelButton.show(); 
            } else {
                this.cancelButton.hide(); 
            }
            if (this.args.showClose) { this.closeButton.show();             } else {
                this.closeButton.hide();
            }
            this.nextButton.show();
            this.backButton.show();
            return this;
        },

        getCard: function(el) {
            var cardDOMEl = $(el).parents(".wizard-card").first()[0];
            if (cardDOMEl) {
                var foundCard = null;
                this.eachCard(function(i, card) {
                    if (cardDOMEl == card.el[0]) {
                        foundCard = card;
                        return false;
                    }
                    return true;
                });
                return foundCard;
            }
            else {
                return null;
            }
        },

        _createCards: function() {
            var prev = null;
            var next = null;
            var currentCard = null;
            var wizard = this;
            var self = this;
            
            self.log("Creating Cards");

            var cards = this.modal.find(".wizard-cards .wizard-card");
            $.each(cards, function(i, card) {
                card = $(card);

                prev = currentCard;
                currentCard = new WizardCard(wizard, card, i, prev, next);
                self._cards.push(currentCard);
                if (currentCard.name) {
                    self.cards[currentCard.name] = currentCard;
                }
                if (prev) {prev.next = currentCard;}

                self.modal.find(".wizard-steps .wizard-nav-list").append(currentCard.nav);
            });
        },

        showSubmitCard: function(name) {
            this.log("showing "+name+" submit card");

            var card = this.el.find(".wizard-"+name);
            if (card.length) {
                this.hideCards();
                this.el.find(".wizard-"+name).show();
            }
            else {
                this.log("couldn't find submit card "+name);
            }
        },

        hideSubmitCard: function(name) {
            this.log("hiding "+name+" submit card");
            this.el.find(".wizard-"+name).hide();
        },

        hideSubmitCards: function() {
            var wizard = this;
            $.each(["success", "error", "failure", "loading"], function(i, name) {
                wizard.hideSubmitCard(name);
            });
        },

        enableNextButton: function() {
            this.log("enabling next button");
            this.nextButton.removeAttr("disabled");
            return this;
        },

        disableNextButton: function() {
            this.log("disabling next button");
            this.nextButton.attr("disabled", "disabled");
            return this;
        },

        serializeArray: function() {
            var form = this.form.serializeArray();
            this.form.find('input[disabled][data-serialize="1"]').each(function() {
                formObj = {
                    name: $(this).attr('name'),
                    value: $(this).val()
                };
                
                form.push(formObj);
            });
            
            return form;
        },

        serialize: function() {
            var form = this.form.serialize();
            this.form.find('input[disabled][data-serialize="1"]').each(function() {
                form = form + '&' + $(this).attr('name') + '=' + $(this).val();
            });
            
            return form;
        },
        
        find: function(selector) {
            return this.modal.find(selector);
        },


        /*
         * the next 3 functions are to be called by the custom submit event
         * handler.  the idea is that after you make an ajax call to submit
         * your wizard data (or whatever it is you want to do at the end of
         * the wizard), you call one of these 3 handlers to display a specific
         * card for either success, failure, or error
         */
        submitSuccess: function() {
            this.log("submit success");
            this._submitting = false;
            this.showSubmitCard("success");
            this.trigger("submitSuccess");
        },

        submitFailure: function() {
            this.log("submit failure");
            this._submitting = false;
            this.showSubmitCard("failure");
            this.trigger("submitFailure");
        },

        submitError: function() {
            this.log("submit error");
            this._submitting = false;
            this.showSubmitCard("error");
            this.trigger("submitError");
        },


        _submit: function() {
            this.log("submitting wizard");
            this._submitting = true;

            this.lockCards();
            this.cancelButton.hide();
            this.closeButton.hide();
            this.backButton.hide();

            this.showSubmitCard("loading");
            this.updateProgressBar(100);

            this.changeNextButton(this.args.buttons.submittingText, false);
            this.disableNextButton();

            var ret = this.trigger("submit");
            this.trigger("loading");
        },

        _onNextClick: function() {
            this.log("handling 'next' button click");
            var currentCard = this.getActiveCard();
            if (this._readyToSubmit && currentCard.validate()) {
                this._submit();
            }
            else {
                currentCard = this.incrementCard();
            }
        },

        _onBackClick: function() {
            this.log("handling 'back' button click");
            var currentCard = this.decrementCard();
        },

        _handleNextClick: function(event) {
            var wizard = event.data;
            wizard._onNextClick.call(wizard);
        },

        _handleBackClick: function(event) {
            var wizard = event.data;
            wizard._onBackClick.call(wizard);
        },


        /*
         * this function is attached by default to the wizard's "submit" event.
         * if you choose to implement your own custom submit logic, you should
         * copy how this function works
         */
        _defaultSubmit: function(wizard) {
            $.ajax({
                type: "POST",
                url: wizard.args.submitUrl,
                data: wizard.serialize(),
                dataType: "json"
            }).done(function(response) {
                wizard.submitSuccess();
                wizard.hideButtons();
                wizard.updateProgressBar(0);
            }).fail(function() {
                wizard.submitFailure();
                wizard.hideButtons();
            });
        }
    };
    
    
}(window.jQuery));
