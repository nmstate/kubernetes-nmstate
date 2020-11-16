FROM ruby
COPY Makefile Gemfile* /docs/
RUN make -C docs install
COPY . /docs/
RUN make -C docs check
