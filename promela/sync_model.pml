/* Modelado inicial en Promela */
#define NUM_WORKERS 3
#define TOTAL_JOBS  6

int jobs_remaining = TOTAL_JOBS;

/* Mutex que protege el slice compartido "forest" */
bit mutex = 0;              /* 0 = libre, 1 = tomado */

/* Estado del forest y de la seccion critica */
int trees_built  = 0;       /* arboles ya insertados en el forest */
int in_critical  = 0;       /* workers dentro de la seccion critica en este instante */

/* WaitGroup cuenta workers que aun no han hecho  */
int wg_counter = NUM_WORKERS;

active [NUM_WORKERS] proctype Worker() {
  bool got_job;

  do
  :: atomic {
       if
       :: jobs_remaining > 0 ->
            jobs_remaining--;
            got_job = true;
       :: else ->
            got_job = false;
       fi
     };

     if
     :: got_job ->
          skip;
          atomic { mutex == 0 -> mutex = 1 }
          in_critical++;
          assert(in_critical == 1);   /* nunca dos workers a la vez, o sea sin race */
          trees_built++;
          in_critical--;
          mutex = 0;                 
     :: !got_job -> break
     fi
  od;
  wg_counter--;
}

/* El programa termina correctamente solo si no quedan jobs, el forest tiene TOTAL_JOBS 
arboles y el WaitGroup llego a 0 */
ltl sin_deadlock     { <> [] (wg_counter == 0) }
ltl forest_completo  { <> [] (trees_built == TOTAL_JOBS) }
ltl exclusion_mutua  { [] (in_critical <= 1) }
